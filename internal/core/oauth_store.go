package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Stored only as a secret value, never as configuration. A single replacement
// persists rotated refresh and access tokens together in the selected backend.
type googleCredential struct {
	RequestedScopes string    `json:"requested_scopes,omitempty"`
	GrantedScopes   string    `json:"granted_scopes,omitempty"`
	Failure         string    `json:"failure,omitempty"`
	Provider        string    `json:"provider"`
	ClientID        string    `json:"client_id"`
	ClientSecret    string    `json:"client_secret"`
	RefreshToken    string    `json:"refresh_token"`
	AccessToken     string    `json:"access_token,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
}

var errOAuthChanged = errors.New("OAuth credential changed during renewal; retry with current credentials")

func parseGoogleCredential(raw string) (googleCredential, error) {
	var c googleCredential
	if len(raw) > 8192 || decode([]byte(raw), &c) != nil || c.Provider != "google" || !oauthValue(c.ClientID) || !oauthValue(c.ClientSecret) || !oauthValue(c.RefreshToken) {
		return googleCredential{}, errOAuthCredentials
	}
	if c.AccessToken != "" && (!oauthValue(c.AccessToken) || c.ExpiresAt.IsZero()) {
		return googleCredential{}, errOAuthCredentials
	}
	if _, err := oauthScopeSet(c.RequestedScopes); err != nil {
		return googleCredential{}, errOAuthCredentials
	}
	if _, err := oauthScopeSet(c.GrantedScopes); err != nil {
		return googleCredential{}, errOAuthCredentials
	}
	switch c.Failure {
	case "", "reconnect_required", "client_rejected", "temporary_error", "invalid_response", "token_rejected", "scopes_missing":
	default:
		return googleCredential{}, errOAuthCredentials
	}
	return c, nil
}

type googleRefresh func(context.Context, string, string, string) (oauthToken, error)

func (e *Engine) googleAccessToken(ctx context.Context, id string) (string, error) {
	return e.googleAccessTokenUsing(ctx, id, time.Now, refreshGoogleToken)
}

func (e *Engine) googleAccessTokenUsing(ctx context.Context, id string, now func() time.Time, refresh googleRefresh) (string, error) {
	// Serialize refreshes, but release mutation during network IO so deletion and
	// revocation remain responsive. Re-read the secret instead of keeping a cache.
	e.oauthMu.Lock()
	defer e.oauthMu.Unlock()
	if ctx.Err() != nil {
		return "", errOAuthTemporary
	}
	e.mutation.Lock()
	raw, err := e.secret(ctx, id)
	e.mutation.Unlock()
	if err != nil {
		return "", errOAuthCredentials
	}
	c, err := parseGoogleCredential(raw)
	if err != nil {
		return "", err
	}
	current := now()
	if c.Failure == "scopes_missing" {
		return "", errOAuthScopes
	}
	if c.Failure == "reconnect_required" {
		return "", errOAuthReconnect
	}
	if c.Failure == "client_rejected" {
		return "", errOAuthClient
	}
	if c.AccessToken != "" && c.Failure == "" && c.ExpiresAt.After(current.Add(60*time.Second)) && !c.ExpiresAt.After(current.Add(24*time.Hour)) {
		return c.AccessToken, nil
	}
	token, refreshErr := refresh(ctx, c.ClientID, c.ClientSecret, c.RefreshToken)
	if refreshErr == nil && token.Scope != "" && !oauthScopesInclude(token.Scope, strings.Fields(c.RequestedScopes)) {
		refreshErr = errOAuthScopes
	}
	if refreshErr != nil {
		switch refreshErr {
		case errOAuthScopes:
			c.Failure = "scopes_missing"
		case errOAuthReconnect:
			c.Failure = "reconnect_required"
		case errOAuthClient:
			c.Failure = "client_rejected"
		case errOAuthTemporary:
			c.Failure = "temporary_error"
		default:
			c.Failure = "invalid_response"
			refreshErr = errOAuthResponse
		}
	} else {
		if !oauthValue(token.AccessToken) || !oauthValue(token.RefreshToken) || !token.ExpiresAt.After(now()) {
			return "", errOAuthResponse
		}
		c.AccessToken, c.RefreshToken, c.ExpiresAt = token.AccessToken, token.RefreshToken, token.ExpiresAt
		if token.Scope != "" {
			c.GrantedScopes = token.Scope
		}
		c.Failure = ""
	}
	updated, err := json.Marshal(c)
	if err != nil || len(updated) > 8192 {
		return "", errOAuthResponse
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	latest, err := e.secret(ctx, id)
	if err != nil || latest != raw {
		return "", errOAuthChanged
	}
	var backend string
	if e.db.QueryRowContext(ctx, "SELECT backend FROM secrets WHERE id=?", id).Scan(&backend) != nil {
		return "", errOAuthCredentials
	}
	if _, err = e.putSecretLocked(ctx, secretWrite{ID: id, Value: string(updated), Backend: backend}); err != nil {
		return "", errOAuthCredentials
	}
	if refreshErr != nil {
		return "", refreshErr
	}
	return token.AccessToken, nil
}
