package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGoogleDestinationBoundary(t *testing.T) {
	for _, raw := range []string{"https://www.googleapis.com/calendar/v3/calendars/primary/events", "https://calendar.googleapis.com:443/v3/test"} {
		if validateGoogleDestination(Destination{URL: raw}) != nil {
			t.Fatal("approved host rejected")
		}
	}
	for _, raw := range []string{"http://www.googleapis.com/x", "https://www.googleapis.com:8443/x", "https://www.googleapis.com.evil.example/x", "https://evil.example/?host=www.googleapis.com", "https://storage.googleapis.com/x", "https://www.googleapis.com./x", "https://user@www.googleapis.com/x", "https://www.googleapis.com/x#fragment", "https://127.0.0.1/x"} {
		if validateGoogleDestination(Destination{URL: raw}) == nil {
			t.Fatalf("unsafe host accepted: %s", raw)
		}
	}
	d := Destination{ID: "google", URL: "https://www.googleapis.com/calendar/v3/test", Method: "POST", Auth: "oauth2-google", Secret: "account"}
	c := EmptyConfig()
	c.Destinations = []Destination{d}
	c.Actions = []Action{{ID: "send", Kind: "http", Destination: "google", Body: `{}`}}
	c.Flows = []Flow{{ID: "flow", Source: "local:test", Enabled: true, Steps: []string{"send"}}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	before := canonicalHash(capabilities(c))
	c.Destinations[0].Secret = "another-account"
	if before == canonicalHash(capabilities(c)) {
		t.Fatal("credential reference not bound to permission")
	}
	c.Destinations[0].PrivateHosts = []string{"www.googleapis.com:443"}
	if c.Validate() == nil {
		t.Fatal("LAN bypass accepted")
	}
}

func TestGoogleOutboundUsesPrivateTokenAndFailsClosed(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := googleCredential{Provider: "google", ClientID: "client", ClientSecret: "private-client-secret", RefreshToken: "private-refresh", AccessToken: "private-access", ExpiresAt: time.Now().Add(time.Hour)}
	save := func() {
		t.Helper()
		raw, _ := json.Marshal(c)
		req, _ := json.Marshal(secretWrite{ID: "account", Backend: "file", Value: string(raw)})
		if _, err := e.putSecret(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
	save()
	d := Destination{ID: "google", URL: "https://www.googleapis.com/calendar/v3/test", Method: "POST", Auth: "oauth2-google", Secret: "account"}
	calls := 0
	client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer private-access" {
			t.Fatal("wrong bearer token")
		}
		if r.Header.Get("Idempotency-Key") != "job-key" {
			t.Fatal("delivery identity lost")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
	})}
	result := e.sendWithClient(ctx, client, d, "job-key", []byte(`{}`))
	if result.Status != 200 || calls != 1 {
		t.Fatal(result)
	}
	generic := d
	generic.Auth = "bearer"
	generic.URL = "https://attacker.example/"
	result = e.sendWithClient(ctx, client, generic, "job-key", []byte(`{}`))
	if calls != 1 || result.Status != 0 {
		t.Fatal("OAuth secret bundle sent as generic bearer")
	}
	c.Failure = "reconnect_required"
	save()
	result = e.sendWithClient(ctx, client, d, "job-key", []byte(`{}`))
	if result.Retry || result.Status != 0 || calls != 1 || strings.Contains(result.Message, "private-") {
		t.Fatal("terminal failure sent or leaked credentials", result)
	}
	d.URL = "https://attacker.example/x"
	result = e.sendWithClient(ctx, client, d, "job-key", []byte(`{}`))
	if result.Retry || calls != 1 || result.Message != errOAuthDestination.Error() {
		t.Fatal("unapproved destination contacted")
	}
}
