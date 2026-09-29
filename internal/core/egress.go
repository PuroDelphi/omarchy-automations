package core

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

var errDestinationBlocked = errors.New("destination address blocked by policy")

var restrictedNetworks = []string{"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4", "::/128", "::1/128", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/32", "2001:db8::/32", "2002::/16", "fc00::/7", "fe80::/10", "ff00::/8"}

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() {
		return false
	}
	for _, raw := range restrictedNetworks {
		if netip.MustParsePrefix(raw).Contains(ip) {
			return false
		}
	}
	return true
}
func allowedPrivate(d Destination, address string) bool {
	return validatePrivateEndpoints(d) == nil && len(d.PrivateHosts) == 1 && d.PrivateHosts[0] == address
}

func outboundClient(d Destination, r resolver) *http.Client {
	transport := &http.Transport{
		Proxy:             nil, // Environmental proxies must not bypass address policy.
		TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
		DisableKeepAlives: true, MaxResponseHeaderBytes: 16384, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 10 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := r.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, errors.New("destination DNS lookup failed")
			}
			if len(ips) == 0 {
				return nil, errors.New("destination has no addresses")
			}
			for _, ip := range ips {
				if !publicAddress(ip) && !(privateUnicast(ip) && allowedPrivate(d, address)) {
					return nil, errDestinationBlocked
				}
			}
			var last error
			for _, ip := range ips {
				conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
				if err == nil {
					return conn, nil
				}
				last = err
			}
			return nil, last
		},
	}
	return &http.Client{Transport: transport, Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}

type deliveryResult struct {
	Status  int
	Retry   bool
	After   time.Duration
	Message string
}

func (e *Engine) send(ctx context.Context, d Destination, key string, body []byte) deliveryResult {
	if e.deliverySender != nil {
		return e.deliverySender(ctx, d, key, body)
	}
	client := outboundClient(d, net.DefaultResolver)
	defer client.CloseIdleConnections()
	return e.sendWithClient(ctx, client, d, key, body)
}
func (e *Engine) sendWithClient(ctx context.Context, client *http.Client, d Destination, key string, body []byte) deliveryResult {
	u, err := url.Parse(d.URL)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return deliveryResult{Message: "invalid HTTPS destination"}
	}
	req, err := http.NewRequestWithContext(ctx, d.Method, d.URL, bytes.NewReader(body))
	if err != nil {
		return deliveryResult{Message: "invalid outbound request"}
	}
	contentType := "application/json"
	if d.Format == "form" {
		contentType = "application/x-www-form-urlencoded"
	} else if d.Format == "raw" {
		contentType = "text/plain; charset=utf-8"
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range d.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Idempotency-Key", key)
	if err := e.authenticateOutbound(ctx, req, d, key, body); err != nil {
		if d.Auth == "oauth2-google" {
			return deliveryResult{Retry: errors.Is(err, errOAuthTemporary) || errors.Is(err, errOAuthChanged), Message: err.Error()}
		}
		return deliveryResult{Retry: true, Message: "destination credential unavailable"}
	}

	var oauthRevision string
	if d.Auth == "oauth2-google" {
		oauthRevision, err = e.googleDispatchRevision(ctx, d.Secret, strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			return deliveryResult{Retry: errors.Is(err, errOAuthChanged), Message: err.Error()}
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return deliveryResult{Retry: true, Message: "outbound transport failed"}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 65536))
	if d.Auth == "oauth2-google" && resp.StatusCode == http.StatusUnauthorized {
		message := "HTTP 401; OAuth token rejected; delivery not repeated"
		if err := e.rejectGoogleToken(ctx, d.Secret, oauthRevision); err != nil {
			message = "HTTP 401; OAuth rejection state could not be saved; delivery not repeated"
		}
		return deliveryResult{Status: resp.StatusCode, Message: message}
	}
	result := deliveryResult{Status: resp.StatusCode, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return result
	}
	result.Retry = resp.StatusCode == 408 || resp.StatusCode == 429 || resp.StatusCode >= 500
	result.After = retryAfter(resp.Header.Get("Retry-After"), time.Now())
	return result
}

func renderHTTP(c Config, a Action, ev Event) ([]byte, error) {
	format := "json"
	for _, d := range c.Destinations {
		if d.ID == a.Destination && d.Format != "" {
			format = d.Format
			break
		}
	}
	if format == "raw" {
		s, err := render(a.Body, ev)
		return []byte(s), err
	}
	b, err := renderJSON(a.Body, ev)
	if err != nil {
		return nil, err
	}
	if format != "form" {
		return b, nil
	}
	var data map[string]any
	if err = json.Unmarshal(b, &data); err != nil {
		return nil, errors.New("form body must be an object")
	}
	values := url.Values{}
	for key, v := range data {
		switch value := v.(type) {
		case string:
			values.Set(key, value)
		case float64, bool:
			values.Set(key, fmt.Sprint(value))
		default:
			return nil, errors.New("form fields must be scalar")
		}
	}
	return []byte(values.Encode()), nil
}
func retryAfter(raw string, now time.Time) time.Duration {
	raw = strings.TrimSpace(raw)
	var delay time.Duration
	if sec, err := strconv.Atoi(raw); err == nil {
		if sec > 3600 {
			sec = 3600
		}
		if sec > 0 {
			delay = time.Duration(sec) * time.Second
		}
	} else if date, err := http.ParseTime(raw); err == nil {
		delay = date.Sub(now)
	}
	if delay < 0 {
		return 0
	}
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

func (e *Engine) authenticateOutbound(ctx context.Context, req *http.Request, d Destination, key string, body []byte) error {
	if d.Auth == "oauth2-google" {
		if err := validateGoogleDestination(d); err != nil {
			return err
		}
		if req.URL.String() != d.URL {
			return errOAuthDestination
		}
		token, err := e.googleAccessToken(ctx, d.Secret)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
	if d.Auth != "" {
		secret, err := e.secret(ctx, d.Secret)
		if err != nil {
			return errors.New("destination credential unavailable")
		}
		if _, err := parseGoogleCredential(secret); err == nil {
			return errors.New("OAuth connection requires Google OAuth authentication")
		}
		if d.Auth == "bearer" {
			req.Header.Set("Authorization", "Bearer "+secret)
		} else {
			stamp := strconv.FormatInt(time.Now().Unix(), 10)
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(stamp + "." + key + "."))
			mac.Write(body)
			req.Header.Set("X-Quatrro-Timestamp", stamp)
			req.Header.Set("X-Quatrro-Delivery", key)
			req.Header.Set("X-Quatrro-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
	}
	return nil
}
