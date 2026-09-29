package core

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
)

func TestLANEndpointValidationAndPermissionBinding(t *testing.T) {
	for _, tc := range []struct{ url, endpoint string }{{"https://nas.local/x", "nas.local:443"}, {"https://nas.local:8443/x", "nas.local:8443"}, {"https://[::1]:8443/x", "[::1]:8443"}} {
		d := Destination{ID: "nas", URL: tc.url, Method: "POST", PrivateHosts: []string{tc.endpoint}}
		if err := validatePrivateEndpoints(d); err != nil {
			t.Fatal(err)
		}
		c := EmptyConfig()
		c.Destinations = []Destination{d}
		c.Actions = []Action{{ID: "send", Kind: "http", Destination: "nas", Body: `{}`}}
		c.Flows = []Flow{{ID: "flow", Source: "local:test", Enabled: true, Steps: []string{"send"}}}
		if err := c.Validate(); err != nil {
			t.Fatal(err)
		}
		with := canonicalHash(capabilities(c))
		c.Destinations[0].PrivateHosts = nil
		if with == canonicalHash(capabilities(c)) {
			t.Fatal("LAN grant not bound to action permission")
		}
	}
	for _, endpoint := range []string{"*", "*.local:443", "other.local:443", "nas.local:8443", "nas.local", "nas.local:0443"} {
		if validatePrivateEndpoints(Destination{URL: "https://nas.local/", PrivateHosts: []string{endpoint}}) == nil {
			t.Fatal("unrelated or ambiguous exception accepted")
		}
	}
	if validatePrivateEndpoints(Destination{URL: "https://nas.local/", PrivateHosts: []string{"nas.local:443", "nas.local:443"}}) == nil {
		t.Fatal("duplicate endpoint accepted")
	}
	for _, raw := range []string{"0.0.0.0", "::", "224.0.0.1", "ff02::1", "192.0.2.1", "2001:db8::1", "64:ff9b::a00:1"} {
		if privateUnicast(netip.MustParseAddr(raw)) {
			t.Fatal("non-LAN range granted", raw)
		}
	}
	for _, raw := range []string{"127.0.0.1", "::1", "::ffff:127.0.0.1", "192.168.1.2", "fd00::1", "100.64.0.1"} {
		if !privateUnicast(netip.MustParseAddr(raw)) {
			t.Fatal("LAN address rejected", raw)
		}
	}
}

func TestLANIPv4IPv6ExactEndpointTransport(t *testing.T) {
	for _, address := range []string{"127.0.0.1:0", "[::1]:0"} {
		t.Run(address, func(t *testing.T) {
			listener, err := net.Listen("tcp", address)
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
			server.Listener.Close()
			server.Listener = listener
			server.StartTLS()
			defer server.Close()
			authority := strings.TrimPrefix(server.URL, "https://")
			d := Destination{URL: server.URL, PrivateHosts: []string{authority}}
			request := func(d Destination) error {
				client := outboundClient(d, net.DefaultResolver)
				defer client.CloseIdleConnections()
				roots := x509.NewCertPool()
				roots.AddCert(server.Certificate())
				client.Transport.(*http.Transport).TLSClientConfig.RootCAs = roots
				resp, err := client.Get(server.URL)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode != 204 {
						t.Fatal(resp.StatusCode)
					}
				}
				return err
			}
			if err = request(d); err != nil {
				t.Fatal("authorized endpoint failed", err)
			}
			d.PrivateHosts = nil
			if request(d) == nil {
				t.Fatal("ungranted LAN reached")
			}
			d.PrivateHosts = []string{"other.local:443"}
			if request(d) == nil {
				t.Fatal("other endpoint grant reached LAN")
			}
			d.PrivateHosts = []string{authority}
			d.URL = "https://other.local/"
			if request(d) == nil {
				t.Fatal("grant reused by other destination")
			}
			if calls.Load() != 1 {
				t.Fatal("blocked request reached server")
			}
		})
	}
}

func TestLANRevocationBlocksQueuedAndRetryDelivery(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(map[bool]string{false: "queued", true: "retry"}[retry], func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(503) }))
			defer server.Close()
			d := Destination{ID: "lan", URL: server.URL, Method: "POST", PrivateHosts: []string{strings.TrimPrefix(server.URL, "https://")}}
			c := EmptyConfig()
			c.Destinations = []Destination{d}
			c.Actions = []Action{{ID: "send", Kind: "http", Destination: "lan", Body: `{}`}}
			c.Flows = []Flow{{ID: "flow", Source: "local:test", Enabled: true, Steps: []string{"send"}}}
			activateTest(t, e, c)
			client := outboundClient(d, net.DefaultResolver)
			defer client.CloseIdleConnections()
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			client.Transport.(*http.Transport).TLSClientConfig.RootCAs = roots
			e.deliverySender = func(ctx context.Context, d Destination, key string, body []byte) deliveryResult {
				return e.sendWithClient(ctx, client, d, key, body)
			}
			if _, err := e.ingest(ctx, Event{Source: "local:test", Type: "test"}); err != nil {
				t.Fatal(err)
			}
			expected := int32(0)
			if retry {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
				expected = 1
				if calls.Load() != expected {
					t.Fatal("initial LAN delivery not reached")
				}
			}
			if _, err := e.revoke(ctx, []byte(`{"scope":"flow:flow:action:send"}`)); err != nil {
				t.Fatal(err)
			}
			if retry {
				if _, err := e.db.Exec("UPDATE outbox SET next_at=0"); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 3; i++ {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var state string
			if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil {
				t.Fatal(err)
			}
			if state != "denied" || calls.Load() != expected {
				t.Fatalf("revocation failed: state=%s calls=%d", state, calls.Load())
			}
		})
	}
}
