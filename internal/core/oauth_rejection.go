package core

import (
	"context"
	"encoding/json"
	"time"
)

// Bind remote rejection to the exact private credential used at dispatch. A
// late 401 must not invalidate a replacement or a concurrently refreshed token.
func (e *Engine) googleDispatchRevision(ctx context.Context, id, token string) (string, error) {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	raw, err := e.secret(ctx, id)
	if err != nil {
		return "", errOAuthCredentials
	}
	c, err := parseGoogleCredential(raw)
	if err != nil || c.AccessToken != token || c.Failure != "" {
		return "", errOAuthChanged
	}
	return canonicalHash(raw), nil
}

func (e *Engine) rejectGoogleToken(ctx context.Context, id, revision string) error {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	raw, err := e.secret(ctx, id)
	if err != nil {
		return errOAuthCredentials
	}
	if canonicalHash(raw) != revision {
		return nil
	}
	c, err := parseGoogleCredential(raw)
	if err != nil {
		return err
	}
	c.AccessToken = ""
	c.ExpiresAt = time.Time{}
	c.Failure = "token_rejected"
	updated, err := json.Marshal(c)
	if err != nil {
		return errOAuthResponse
	}
	var backend string
	if e.db.QueryRowContext(ctx, "SELECT backend FROM secrets WHERE id=?", id).Scan(&backend) != nil {
		return errOAuthCredentials
	}
	if _, err = e.putSecretLocked(ctx, secretWrite{ID: id, Value: string(updated), Backend: backend}); err != nil {
		return errOAuthCredentials
	}
	return nil
}
