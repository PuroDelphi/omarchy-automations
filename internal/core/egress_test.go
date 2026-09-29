package core

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type staticResolver struct{ ips []netip.Addr }

func (s staticResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return s.ips, nil
}

func TestOutboundFormAndRawEncoding(t *testing.T) {
	c := EmptyConfig()
	c.Destinations = []Destination{{ID: "out", Format: "form"}}
	a := Action{Destination: "out", Body: `{"message":"{{data.message}}"}`}
	ev := Event{Data: map[string]any{"message": "a&admin=true"}}
	b, err := renderHTTP(c, a, ev)
	if err != nil {
		t.Fatal(err)
	}
	form, _ := url.ParseQuery(string(b))
	if len(form) != 1 || form.Get("message") != "a&admin=true" {
		t.Fatal("form injection", string(b))
	}
	c.Destinations[0].Format = "raw"
	a.Body = "{{data.message}}"
	b, err = renderHTTP(c, a, ev)
	if err != nil || string(b) != "a&admin=true" {
		t.Fatal("raw body", string(b), err)
	}
}
func TestAddressPolicy(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::ffff:127.0.0.1", "::1", "fc00::1", "64:ff9b::7f00:1", "100.64.0.1", "224.0.0.1"} {
		if publicAddress(netip.MustParseAddr(ip)) {
			t.Error("allowed", ip)
		}
	}
	if !publicAddress(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("blocked public address")
	}
}
func TestHTTPSDeliveryUsesValidatedAddressAndNoRedirect(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Idempotency-Key") != "job-1" {
			t.Error("missing stable key")
		}
		w.Header().Set("Location", "https://127.0.0.1/secret")
		w.WriteHeader(302)
	}))
	defer server.Close()
	d := Destination{ID: "test", URL: server.URL, Method: "POST"}
	client := outboundClient(d, net.DefaultResolver)
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("loopback not blocked")
	}
	address := strings.TrimPrefix(server.URL, "https://")
	d.PrivateHosts = []string{address}
	client = outboundClient(d, net.DefaultResolver)
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	client.Transport.(*http.Transport).TLSClientConfig.RootCAs = roots
	e := testEngine(t)
	result := e.sendWithClient(context.Background(), client, d, "job-1", []byte(`{}`))
	if result.Status != 302 || result.Retry || calls.Load() != 1 {
		t.Fatal(result, calls.Load())
	}
	// Host resolves to mixed public/private addresses; reject before making any connection.
	client = outboundClient(Destination{}, staticResolver{[]netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}})
	_, err = client.Get("https://example.test")
	if err == nil {
		t.Fatal("mixed DNS addresses accepted")
	}
}
func TestRetryAfterBounded(t *testing.T) {
	if retryAfter("9999999", time.Now()) != time.Hour {
		t.Fatal("unbounded retry-after")
	}
	if retryAfter("-4", time.Now()) != 0 {
		t.Fatal("negative retry-after")
	}
}

type changingResolver struct{ calls atomic.Int32 }

func (r *changingResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	if r.calls.Add(1) == 1 {
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	return []netip.Addr{netip.MustParseAddr("224.0.0.1")}, nil
}

func TestDNSChangesRevalidatedWithoutSecondLookup(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(204)
	}))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	authority := net.JoinHostPort("changing.example", endpoint.Port())
	d := Destination{URL: "https://" + authority, PrivateHosts: []string{authority}}
	resolver := &changingResolver{}
	client := outboundClient(d, resolver)
	defer client.CloseIdleConnections()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig.RootCAs = roots
	// Fixture certificate authenticates the local test receiver; production
	// clients retain automatic URL hostname verification.
	transport.TLSClientConfig.ServerName = "127.0.0.1"
	response, err := client.Get(d.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 204 || resolver.calls.Load() != 1 || received.Load() != 1 {
		t.Fatal("validated address not used directly", resolver.calls.Load(), received.Load())
	}
	_, err = client.Get(d.URL)
	if !errors.Is(err, errDestinationBlocked) || resolver.calls.Load() != 2 || received.Load() != 1 {
		t.Fatal("changed DNS bypassed validation", err, resolver.calls.Load(), received.Load())
	}
}
