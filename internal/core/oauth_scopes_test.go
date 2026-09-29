package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGoogleConsentRequiresGrantedScopes(t *testing.T) {
	required := "https://www.googleapis.com/auth/calendar.events"
	for _, grant := range []string{"", "https://www.googleapis.com/auth/calendar.readonly", required, required + " https://www.googleapis.com/auth/calendar.readonly"} {
		a, _, err := newGoogleAuthorization("client", "secret", "http://127.0.0.1:45678/oauth/callback", []string{required}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		code, err := a.consumeCallback(httptest.NewRequest("GET", a.redirect+"?"+url.Values{"state": {a.state}, "code": {"code"}}.Encode(), nil), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		client := &http.Client{Transport: oauthTransport(func(*http.Request) (*http.Response, error) {
			raw, _ := json.Marshal(map[string]any{"access_token": "access", "refresh_token": "refresh", "token_type": "Bearer", "expires_in": 3600, "scope": grant})
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
		})}
		token, err := a.exchange(context.Background(), client, code, time.Now())
		if strings.Contains(grant, required) {
			if err != nil || token.Scope != grant {
				t.Fatal("granted scope lost", err)
			}
		} else if err != errOAuthScopes || token != (oauthToken{}) {
			t.Fatal("partial consent accepted")
		}
	}
	for _, scope := range []string{"a a", "a\tb", "a\nb", "a  b", "a\\b", "\"scope\"", strings.Repeat("x", 513)} {
		if _, err := oauthScopeSet(scope); err == nil {
			t.Fatal("invalid scope encoding accepted")
		}
	}
}

func TestGoogleRefreshScopeReductionStopsUse(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	required := "https://www.googleapis.com/auth/calendar.events"
	c := googleCredential{Provider: "google", ClientID: "client", ClientSecret: "secret", RefreshToken: "refresh", RequestedScopes: required, GrantedScopes: required}
	raw, _ := json.Marshal(c)
	req, _ := json.Marshal(secretWrite{ID: "account", Backend: "file", Value: string(raw)})
	if _, err := e.putSecret(ctx, req); err != nil {
		t.Fatal(err)
	}
	calls := 0
	refresh := func(context.Context, string, string, string) (oauthToken, error) {
		calls++
		return oauthToken{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Hour), Scope: "https://www.googleapis.com/auth/calendar.readonly"}, nil
	}
	for i := 0; i < 2; i++ {
		token, err := e.googleAccessTokenUsing(ctx, "account", time.Now, refresh)
		if token != "" || err != errOAuthScopes {
			t.Fatal("scope reduction accepted")
		}
	}
	if calls != 1 {
		t.Fatal("terminal scope failure retried")
	}
	status, err := e.googleConnectionStatus(ctx, []byte(`{"id":"account"}`))
	if err != nil || status.(map[string]any)["state"] != "scopes_missing" {
		t.Fatal("scope failure not visible")
	}
}
