package core

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type oauthTransport func(*http.Request) (*http.Response, error)

func (f oauthTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleOAuthRefreshContract(t *testing.T) {
	now := time.Unix(1000, 0)
	for _, rotated := range []string{"", "rotated-token"} {
		client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != googleTokenEndpoint || r.Method != "POST" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Fatal("wrong endpoint/method/encoding")
			}
			raw, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(raw))
			if form.Get("client_id") != "client" || form.Get("client_secret") != "secret&=+" || form.Get("refresh_token") != "refresh" || form.Get("grant_type") != "refresh_token" || len(form) != 4 {
				t.Fatal("invalid form")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"access","token_type":"Bearer","expires_in":3600,"refresh_token":"` + rotated + `","scope":"ignored metadata"}`)), Header: make(http.Header)}, nil
		})}
		got, err := refreshGoogleTokenWithClient(context.Background(), client, "client", "secret&=+", "refresh", now)
		expected := rotated
		if expected == "" {
			expected = "refresh"
		}
		if err != nil || got.AccessToken != "access" || got.RefreshToken != expected || !got.ExpiresAt.Equal(now.Add(time.Hour)) {
			t.Fatalf("refresh contract failed: %v", err)
		}
	}
}

func TestGoogleOAuthFailureRedaction(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{400, `{"error":"invalid_grant","error_description":"sensitive-token"}`, errOAuthReconnect},
		{401, `{"error":"invalid_client","error_description":"sensitive-token"}`, errOAuthClient},
		{429, `sensitive-token`, errOAuthTemporary},
		{503, `sensitive-token`, errOAuthTemporary},
		{200, `{"access_token":"sensitive-token","expires_in":3600}`, errOAuthResponse},
		{200, `{"access_token":"sensitive-token","token_type":"Bearer","expires_in":0}`, errOAuthResponse},
		{200, `{"access_token":"sensitive-token","token_type":"Bearer","expires_in":86401}`, errOAuthResponse},
		{200, `{"access_token":"bad\r\nheader","token_type":"Bearer","expires_in":3600}`, errOAuthResponse},
		{200, `{} {}`, errOAuthResponse},
		{200, strings.Repeat("x", 32769), errOAuthResponse},
		{302, `{}`, errOAuthResponse},
	}
	for _, tc := range cases {
		calls := 0
		client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": []string{"https://attacker.example/"}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		})}
		got, err := refreshGoogleTokenWithClient(context.Background(), client, "client", "secret", "refresh", time.Now())
		if !errors.Is(err, tc.want) || got != (oauthToken{}) || calls != 1 {
			t.Fatalf("status %d: got %v calls=%d", tc.status, err, calls)
		}
		if strings.Contains(err.Error(), "sensitive-token") {
			t.Fatal("secret leaked")
		}
	}
	client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) { return nil, errors.New("sensitive-token") })}
	_, err := refreshGoogleTokenWithClient(context.Background(), client, "client", "secret", "refresh", time.Now())
	if err != errOAuthTemporary {
		t.Fatal("transport error not redacted")
	}
	_, err = refreshGoogleTokenWithClient(context.Background(), client, "client", "bad\nsecret", "refresh", time.Now())
	if err != errOAuthCredentials {
		t.Fatal("invalid credentials accepted")
	}
}
