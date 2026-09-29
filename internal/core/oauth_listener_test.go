package core

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGoogleConsentListenerLifecycle(t *testing.T) {
	for _, deny := range []bool{false, true} {
		t.Run(map[bool]string{false: "code", true: "denied"}[deny], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			session, err := listenGoogleConsent(ctx, "client", "secret", []string{"https://www.googleapis.com/auth/calendar.events"})
			if err != nil {
				t.Fatal(err)
			}
			defer session.cancel()
			a := session.authorization
			client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil}}
			defer client.CloseIdleConnections()
			response, err := client.Get(a.redirect + "?state=wrong&code=must-not-echo")
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != 400 || strings.Contains(string(body), "must-not-echo") || response.Header.Get("Cache-Control") != "no-store" {
				t.Fatal("malformed callback handling")
			}
			select {
			case <-session.done:
				t.Fatal("invalid request consumed session")
			default:
			}
			q := url.Values{"state": {a.state}}
			if deny {
				q.Set("error", "access_denied")
			} else {
				q.Set("code", "private-code")
			}
			response, err = client.Get(a.redirect + "?" + q.Encode())
			if err != nil {
				t.Fatal(err)
			}
			body, _ = io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != 200 || strings.Contains(string(body), "private-code") {
				t.Fatal("callback response leaked code")
			}
			select {
			case result := <-session.done:
				if deny {
					if result.err != errOAuthConsentDenied {
						t.Fatal("denial lost")
					}
				} else if result.err != nil || result.code != "private-code" {
					t.Fatal("callback lost")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("listener did not finish")
			}
			endpoint, _ := url.Parse(a.redirect)
			conn, err := net.DialTimeout("tcp", endpoint.Host, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				t.Fatal("listener remained open")
			}
		})
	}
}

func TestGoogleConsentListenerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	session, err := listenGoogleConsent(ctx, "client", "secret", []string{"https://www.googleapis.com/auth/calendar.events"})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	select {
	case result := <-session.done:
		if result.err != errOAuthCancelled || result.code != "" {
			t.Fatal("cancel result unsafe")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel did not close listener")
	}
	endpoint, _ := url.Parse(session.authorization.redirect)
	conn, err := net.DialTimeout("tcp", endpoint.Host, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Fatal("cancel left port open")
	}
	if _, err := listenGoogleConsent(ctx, "client", "secret", []string{"https://www.googleapis.com/auth/calendar.events"}); err != errOAuthCancelled {
		t.Fatal("cancelled parent opened listener")
	}
}

func TestGoogleConsentListenerDeadlineClosesIncompleteRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	session, err := listenGoogleConsent(ctx, "client", "secret", []string{"https://www.googleapis.com/auth/calendar.events"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.cancel()
	endpoint, _ := url.Parse(session.authorization.redirect)
	conn, err := net.DialTimeout("tcp", endpoint.Host, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = io.WriteString(conn, "GET /oauth/callback HTTP/1.1\r\nHost: "); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-session.done:
		if result.err != errOAuthCancelled {
			t.Fatal("deadline result incorrect")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("incomplete request held listener open")
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	if _, err = conn.Read(b[:]); err == nil {
		t.Fatal("incomplete connection remained usable")
	} else if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
		t.Fatal("incomplete connection was not closed")
	}
}
