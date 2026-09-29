package core

import (
	"context"
	"encoding/json"
	"time"
)

// Provisioning imports an already-authorized refresh credential. It performs no
// provider request and grants no flow permission. Interactive consent is separate.
func (e *Engine) putGoogleConnection(ctx context.Context, data []byte) (any, error) {
	var req struct {
		ID           string `json:"id"`
		Backend      string `json:"backend"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`
	}
	if decode(data, &req) != nil {
		return nil, errOAuthCredentials
	}
	c := googleCredential{Provider: "google", ClientID: req.ClientID, ClientSecret: req.ClientSecret, RefreshToken: req.RefreshToken}
	value, err := json.Marshal(c)
	if err != nil {
		return nil, errOAuthCredentials
	}
	if _, err = parseGoogleCredential(string(value)); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(secretWrite{ID: req.ID, Backend: req.Backend, Value: string(value)})
	if err != nil {
		return nil, errOAuthCredentials
	}
	if _, err = e.putSecret(ctx, raw); err != nil {
		return nil, errOAuthCredentials
	}
	return map[string]string{"id": req.ID, "provider": "google", "state": "renewal_required"}, nil
}

func (e *Engine) googleConnectionStatus(ctx context.Context, data []byte) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if decode(data, &req) != nil || !identifier.MatchString(req.ID) {
		return nil, errOAuthCredentials
	}
	// Reading status never contacts the provider or initiates a renewal.
	e.mutation.Lock()
	defer e.mutation.Unlock()
	result := map[string]any{"id": req.ID, "provider": "google", "state": "unavailable"}
	raw, err := e.secret(ctx, req.ID)
	if err != nil {
		return result, nil
	}
	c, err := parseGoogleCredential(raw)
	if err != nil {
		return result, nil
	}
	now := time.Now()
	state := "renewal_required"
	if c.AccessToken != "" {
		if !c.ExpiresAt.After(now) {
			state = "expired"
		} else if c.ExpiresAt.After(now.Add(time.Minute)) && !c.ExpiresAt.After(now.Add(24*time.Hour)) {
			state = "available"
		}
		result["expires_at"] = c.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if c.Failure != "" {
		state = c.Failure
	}
	result["requested_scopes"] = c.RequestedScopes
	result["granted_scopes"] = c.GrantedScopes
	result["state"] = state
	return result, nil
}
