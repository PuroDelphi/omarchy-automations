package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGoogleConsentSessionPersistenceAndConflict(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(map[bool]string{false: "save", true: "conflict"}[conflict], func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
				if conflict {
					putGoogleFixture(t, e, "concurrent-refresh")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"access_token":"private-access","refresh_token":"private-refresh","scope":"https://www.googleapis.com/auth/calendar.events","token_type":"Bearer","expires_in":3600}`))}, nil
			})}
			raw := []byte(`{"id":"google-account","backend":"file","client_id":"client","client_secret":"private-client-secret","scopes":["https://www.googleapis.com/auth/calendar.events"]}`)
			value, err := e.startGoogleConsent(ctx, raw, client)
			if err != nil {
				t.Fatal(err)
			}
			start := value.(map[string]string)
			auth, _ := url.Parse(start["authorization_url"])
			q := auth.Query()
			if strings.Contains(start["authorization_url"], "private-client-secret") {
				t.Fatal("secret exposed")
			}
			if _, err = e.startGoogleConsent(ctx, raw, client); err != errOAuthSession {
				t.Fatal("second session admitted")
			}
			response, err := http.Get(q.Get("redirect_uri") + "?" + url.Values{"state": {q.Get("state")}, "code": {"code"}}.Encode())
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			e.consentMu.Lock()
			done := e.consent.done
			e.consentMu.Unlock()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("session did not finish")
			}
			statusRaw, _ := json.Marshal(map[string]string{"session": start["session"]})
			status, err := e.googleConsentStatus(statusRaw, false)
			if err != nil {
				t.Fatal(err)
			}
			want := "completed"
			if conflict {
				want = "credential_changed"
			}
			if status.(map[string]string)["state"] != want {
				t.Fatal(status)
			}
			serialized, _ := json.Marshal(status)
			if strings.Contains(string(serialized), "private-") {
				t.Fatal("status leaked secrets")
			}
			secret, err := e.secret(ctx, "google-account")
			if err != nil {
				t.Fatal(err)
			}
			c, err := parseGoogleCredential(secret)
			if err != nil {
				t.Fatal(err)
			}
			expected := "private-refresh"
			if conflict {
				expected = "concurrent-refresh"
			}
			if c.RefreshToken != expected {
				t.Fatal("credential mismatch")
			}
			var grants int
			if err = e.db.QueryRow("SELECT count(*) FROM grants").Scan(&grants); err != nil || grants != 0 {
				t.Fatal("consent granted flow permissions")
			}
		})
	}
}

func TestGoogleConsentSessionCancelAndClose(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	raw := []byte(`{"id":"account","backend":"file","client_id":"client","client_secret":"secret","scopes":["https://www.googleapis.com/auth/calendar.events"]}`)
	value, err := e.startGoogleConsent(ctx, raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	start := value.(map[string]string)
	req, _ := json.Marshal(map[string]string{"session": start["session"]})
	status, err := e.googleConsentStatus(req, true)
	if err != nil || status.(map[string]string)["state"] != "cancelled" {
		t.Fatal("cancel failed")
	}
	e.closeGoogleConsent()
	if _, err = e.secret(ctx, "account"); err == nil {
		t.Fatal("cancel saved credentials")
	}
	if _, err = e.startGoogleConsent(ctx, raw, nil); err != errOAuthSession {
		t.Fatal("closed engine accepted consent")
	}
}

func TestGoogleConsentCancellationDuringExchange(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	entered := make(chan struct{})
	client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
		close(entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	raw := []byte(`{"id":"account","backend":"file","client_id":"client","client_secret":"secret","scopes":["https://www.googleapis.com/auth/calendar.events"]}`)
	v, err := e.startGoogleConsent(ctx, raw, client)
	if err != nil {
		t.Fatal(err)
	}
	start := v.(map[string]string)
	auth, _ := url.Parse(start["authorization_url"])
	q := auth.Query()
	response, err := http.Get(q.Get("redirect_uri") + "?" + url.Values{"state": {q.Get("state")}, "code": {"code"}}.Encode())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("exchange not started")
	}
	req, _ := json.Marshal(map[string]string{"session": start["session"]})
	if _, err = e.googleConsentStatus(req, true); err != nil {
		t.Fatal(err)
	}
	e.consentMu.Lock()
	done := e.consent.done
	e.consentMu.Unlock()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("exchange did not cancel")
	}
	state, err := e.googleConsentStatus(req, false)
	if err != nil || state.(map[string]string)["state"] != "cancelled" {
		t.Fatal("cancel state lost")
	}
	if _, err = e.secret(ctx, "account"); err == nil {
		t.Fatal("cancelled exchange persisted credential")
	}
}

func TestGoogleConsentDiscoveryAndRestart(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	empty, err := e.currentGoogleConsent([]byte(`{}`))
	if err != nil || empty.(map[string]string)["state"] != "none" {
		t.Fatal("initial discovery")
	}
	raw := []byte(`{"id":"account","backend":"file","client_id":"client","client_secret":"private-secret","scopes":["https://www.googleapis.com/auth/calendar.events"]}`)
	start, err := e.startGoogleConsent(ctx, raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	current, err := e.currentGoogleConsent([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	a, b := start.(map[string]string), current.(map[string]string)
	if a["session"] != b["session"] || a["authorization_url"] != b["authorization_url"] || b["state"] != "waiting" {
		t.Fatal("discovery lost session")
	}
	serialized, _ := json.Marshal(b)
	if strings.Contains(string(serialized), "private-secret") {
		t.Fatal("discovery leaked secret")
	}
	req, _ := json.Marshal(map[string]string{"session": a["session"]})
	if _, err = e.googleConsentStatus(req, true); err != nil {
		t.Fatal(err)
	}
	current, err = e.currentGoogleConsent([]byte(`{}`))
	if err != nil || current.(map[string]string)["authorization_url"] != "" {
		t.Fatal("cancel retained browser URL")
	}
	e.closeGoogleConsent()
	reopened, err := Open(e.paths)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	current, err = reopened.currentGoogleConsent([]byte(`{}`))
	if err != nil || current.(map[string]string)["state"] != "none" {
		t.Fatal("session survived new engine")
	}
	if _, err = reopened.googleConsentStatus(req, false); err != errOAuthSession {
		t.Fatal("old session accepted after restart")
	}
}
