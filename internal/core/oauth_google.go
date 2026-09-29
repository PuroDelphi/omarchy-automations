package core

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const googleTokenEndpoint = "https://oauth2.googleapis.com/token"

var (
	errOAuthCredentials = errors.New("OAuth credentials unavailable or invalid")
	errOAuthReconnect   = errors.New("OAuth authorization expired or invalidated; reconnect account")
	errOAuthClient      = errors.New("OAuth client rejected; check client configuration")
	errOAuthTemporary   = errors.New("OAuth provider temporarily unavailable")
	errOAuthResponse    = errors.New("OAuth provider returned an invalid response")
)

// Private result: never include this object in API responses, audit or exports.
type oauthToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Scope        string
}

func oauthValue(s string) bool {
	if len(s) == 0 || len(s) > 8192 {
		return false
	}
	for _, r := range s {
		if r <= 32 || r >= 127 {
			return false
		}
	}
	return true
}

// The caller owns persistence and serialization of concurrent refreshes. The
// production entry point always uses the existing public-only pinned transport.
func refreshGoogleToken(ctx context.Context, clientID, clientSecret, refreshToken string) (oauthToken, error) {
	client := outboundClient(Destination{}, net.DefaultResolver)
	defer client.CloseIdleConnections()
	return refreshGoogleTokenWithClient(ctx, client, clientID, clientSecret, refreshToken, time.Now())
}

func refreshGoogleTokenWithClient(ctx context.Context, client *http.Client, clientID, clientSecret, refreshToken string, now time.Time) (oauthToken, error) {
	if !oauthValue(clientID) || !oauthValue(clientSecret) || !oauthValue(refreshToken) {
		return oauthToken{}, errOAuthCredentials
	}
	body := url.Values{"client_id": {clientID}, "client_secret": {clientSecret}, "refresh_token": {refreshToken}, "grant_type": {"refresh_token"}}
	return googleTokenRequest(ctx, client, body, refreshToken, now)
}

func googleTokenRequest(ctx context.Context, client *http.Client, body url.Values, refreshToken string, now time.Time) (oauthToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return oauthToken{}, errOAuthCredentials
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	// Clone to prohibit redirects even if the injected client has a permissive policy.
	bounded := *client
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	bounded.Timeout = 15 * time.Second
	resp, err := bounded.Do(req)
	if err != nil {
		return oauthToken{}, errOAuthTemporary
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32769))
	if err != nil || len(raw) > 32768 {
		return oauthToken{}, errOAuthResponse
	}
	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return oauthToken{}, errOAuthTemporary
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Error        string `json:"error"`
		Scope        string `json:"scope"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return oauthToken{}, errOAuthResponse
	}
	if resp.StatusCode != http.StatusOK {
		switch result.Error {
		case "invalid_grant":
			return oauthToken{}, errOAuthReconnect
		case "invalid_client", "unauthorized_client":
			return oauthToken{}, errOAuthClient
		}
		return oauthToken{}, errOAuthResponse
	}
	if result.Error != "" || !oauthValue(result.AccessToken) || !strings.EqualFold(result.TokenType, "Bearer") || result.ExpiresIn < 1 || result.ExpiresIn > 86400 {
		return oauthToken{}, errOAuthResponse
	}
	if result.RefreshToken == "" {
		result.RefreshToken = refreshToken
	}
	if !oauthValue(result.RefreshToken) {
		return oauthToken{}, errOAuthResponse
	}
	if _, err := oauthScopeSet(result.Scope); err != nil {
		return oauthToken{}, errOAuthResponse
	}
	return oauthToken{result.AccessToken, result.RefreshToken, now.Add(time.Duration(result.ExpiresIn) * time.Second), result.Scope}, nil
}
