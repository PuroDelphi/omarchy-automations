package core

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGoogleConsentPKCEAndExchange(t *testing.T) {
	now := time.Now()
	a, authorization, err := newGoogleAuthorization("client", "private-client-secret", "http://127.0.0.1:45678/oauth/callback", []string{"https://www.googleapis.com/auth/calendar.events"}, now)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(authorization)
	q := u.Query()
	hash := sha256.Sum256([]byte(a.verifier))
	if u.Host != "accounts.google.com" || q.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(hash[:]) || q.Get("code_challenge_method") != "S256" || len(a.verifier) != 43 || len(a.state) != 43 || a.state == a.verifier || strings.Contains(authorization, "private-client-secret") || strings.Contains(authorization, a.verifier) {
		t.Fatal("invalid authorization URL or PKCE")
	}
	callback := httptest.NewRequest("GET", a.redirect+"?"+url.Values{"state": {a.state}, "code": {"private-code"}}.Encode(), nil)
	code, err := a.consumeCallback(callback, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.consumeCallback(callback, now); err != errOAuthCallback {
		t.Fatal("callback replay accepted")
	}
	calls := 0
	client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		raw, _ := io.ReadAll(r.Body)
		p, _ := url.ParseQuery(string(raw))
		if p.Get("code_verifier") != a.verifier || p.Get("code") != "private-code" || p.Get("redirect_uri") != a.redirect || p.Get("grant_type") != "authorization_code" {
			t.Fatal("exchange lost session binding")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"access","refresh_token":"refresh","scope":"https://www.googleapis.com/auth/calendar.events","token_type":"Bearer","expires_in":3600}`)), Header: make(http.Header)}, nil
	})}
	token, err := a.exchange(context.Background(), client, code, now)
	if err != nil || token.RefreshToken != "refresh" {
		t.Fatal("exchange failed", err)
	}
	if _, err = a.exchange(context.Background(), client, code, now); err != errOAuthCallback || calls != 1 {
		t.Fatal("code exchange replay accepted")
	}
}

func TestGoogleConsentCallbackValidation(t *testing.T) {
	now := time.Now()
	fresh := func() *googleAuthorization {
		t.Helper()
		a, _, err := newGoogleAuthorization("client", "secret", "http://127.0.0.1:45678/oauth/callback", []string{"https://www.googleapis.com/auth/calendar.events"}, now)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	a := fresh()
	for _, query := range []string{"state=wrong&code=x", "state=" + a.state + "&state=" + a.state + "&code=x", "state=" + a.state + "&code=x&code=y", "state=" + a.state + "&code=x&error=denied", "state=" + a.state + "&code=%00", "state=" + a.state + "&code=x;extra=y"} {
		if _, err := a.consumeCallback(httptest.NewRequest("GET", a.redirect+"?"+query, nil), now); err != errOAuthCallback {
			t.Fatal("invalid callback accepted")
		}
	}
	for _, target := range []string{"http://attacker.example/oauth/callback", "http://127.0.0.1:45678/wrong"} {
		if _, err := a.consumeCallback(httptest.NewRequest("GET", target+"?state="+a.state+"&code=x", nil), now); err != errOAuthCallback {
			t.Fatal("callback origin/path accepted")
		}
	}
	if a.consumed {
		t.Fatal("malformed request consumed consent")
	}
	if _, err := a.consumeCallback(httptest.NewRequest("GET", a.redirect+"?state="+a.state+"&error=access_denied", nil), now); err != errOAuthConsentDenied {
		t.Fatal("consent refusal missing")
	}
	a = fresh()
	if _, err := a.consumeCallback(httptest.NewRequest("GET", a.redirect+"?state="+a.state+"&code=x", nil), now.Add(10*time.Minute)); err != errOAuthCallback {
		t.Fatal("expired callback accepted")
	}
	for _, redirect := range []string{"https://127.0.0.1:123/oauth/callback", "http://localhost:123/oauth/callback", "http://0.0.0.0:123/oauth/callback", "http://127.0.0.1/oauth/callback"} {
		if _, _, err := newGoogleAuthorization("client", "secret", redirect, []string{"https://www.googleapis.com/auth/calendar.events"}, now); err == nil {
			t.Fatal("unsafe redirect accepted")
		}
	}
}

func TestGoogleConsentRequiresOfflineRefreshToken(t *testing.T) {
	now := time.Now()
	a, _, err := newGoogleAuthorization("client", "secret", "http://127.0.0.1:45678/oauth/callback", []string{"https://www.googleapis.com/auth/calendar.events"}, now)
	if err != nil {
		t.Fatal(err)
	}
	code, err := a.consumeCallback(httptest.NewRequest("GET", a.redirect+"?state="+a.state+"&code=code", nil), now)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: oauthTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"access","token_type":"Bearer","expires_in":3600}`)), Header: make(http.Header)}, nil
	})}
	token, err := a.exchange(context.Background(), client, code, now)
	if err != errOAuthResponse || token != (oauthToken{}) {
		t.Fatal("incomplete offline authorization accepted")
	}
}
