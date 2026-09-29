package core

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

type googleConsentSession struct {
	id, connection, state string
	authorizationURL      string
	cancel                context.CancelFunc
	done                  chan struct{}
}

var errOAuthSession = errors.New("OAuth session unavailable or another authorization is pending")

func (e *Engine) startGoogleConsent(ctx context.Context, data []byte, client *http.Client) (any, error) {
	var req struct {
		ID           string   `json:"id"`
		Backend      string   `json:"backend"`
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		Scopes       []string `json:"scopes"`
	}
	if decode(data, &req) != nil || !identifier.MatchString(req.ID) || (req.Backend != "file" && req.Backend != "keyring") {
		return nil, errOAuthCredentials
	}
	e.consentMu.Lock()
	defer e.consentMu.Unlock()
	if e.consentClosed {
		return nil, errOAuthSession
	}
	if s := e.consent; s != nil {
		select {
		case <-s.done:
		default:
			return nil, errOAuthSession
		}
	}
	e.mutation.Lock()
	var count int
	err := e.db.QueryRowContext(ctx, "SELECT count(*) FROM secrets WHERE id=?", req.ID).Scan(&count)
	var original string
	if err == nil && count != 0 {
		original, err = e.secret(ctx, req.ID)
	}
	e.mutation.Unlock()
	if err != nil {
		return nil, errOAuthCredentials
	}
	// The local RPC's lifetime must not cancel browser consent when it returns.
	sessionCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	listener, err := listenGoogleConsent(sessionCtx, req.ClientID, req.ClientSecret, req.Scopes)
	if err != nil {
		cancel()
		return nil, err
	}
	s := &googleConsentSession{id: newID(), connection: req.ID, state: "waiting", authorizationURL: listener.url, cancel: cancel, done: make(chan struct{})}
	e.consent = s
	go func() {
		defer close(s.done)
		defer cancel()
		result := <-listener.done
		state := "failed"
		var token oauthToken
		if result.err == errOAuthCancelled {
			state = "cancelled"
		} else if result.err == errOAuthConsentDenied {
			state = "denied"
		} else if result.err == nil {
			e.consentMu.Lock()
			if sessionCtx.Err() == nil {
				s.state = "exchanging"
				s.authorizationURL = ""
			}
			e.consentMu.Unlock()
			if client == nil {
				client = outboundClient(Destination{}, net.DefaultResolver)
				defer client.CloseIdleConnections()
			}
			token, err = listener.authorization.exchange(sessionCtx, client, result.code, time.Now())
			if err == errOAuthScopes {
				state = "scopes_missing"
			}
			if err == nil {
				state = "ready"
			}
		}
		e.consentMu.Lock()
		defer e.consentMu.Unlock()
		s.authorizationURL = ""
		if sessionCtx.Err() != nil {
			s.state = "cancelled"
			return
		}
		if state != "ready" {
			s.state = state
			return
		}
		// Cancel and final persistence are serialized; late responses cannot undo a
		// completed cancellation. Credential replacement uses compare-and-replace.
		e.mutation.Lock()
		defer e.mutation.Unlock()
		var latestCount int
		if e.db.QueryRowContext(sessionCtx, "SELECT count(*) FROM secrets WHERE id=?", req.ID).Scan(&latestCount) != nil {
			s.state = "failed"
			return
		}
		if latestCount != count {
			s.state = "credential_changed"
			return
		}
		if count != 0 {
			latest, readErr := e.secret(sessionCtx, req.ID)
			if readErr != nil || latest != original {
				s.state = "credential_changed"
				return
			}
		}
		c := googleCredential{RequestedScopes: strings.Join(req.Scopes, " "), GrantedScopes: token.Scope, Provider: "google", ClientID: req.ClientID, ClientSecret: req.ClientSecret, RefreshToken: token.RefreshToken, AccessToken: token.AccessToken, ExpiresAt: token.ExpiresAt}
		raw, marshalErr := json.Marshal(c)
		if marshalErr != nil || len(raw) > 8192 {
			s.state = "failed"
			return
		}
		if _, saveErr := e.putSecretLocked(sessionCtx, secretWrite{ID: req.ID, Backend: req.Backend, Value: string(raw)}); saveErr != nil {
			s.state = "storage_failed"
			return
		}
		s.state = "completed"
	}()
	return map[string]string{"session": s.id, "id": req.ID, "state": "waiting", "authorization_url": listener.url}, nil
}

func (e *Engine) googleConsentStatus(data []byte, cancel bool) (any, error) {
	var req struct {
		Session string `json:"session"`
	}
	if decode(data, &req) != nil {
		return nil, errOAuthSession
	}
	e.consentMu.Lock()
	defer e.consentMu.Unlock()
	s := e.consent
	if s == nil || s.id != req.Session {
		return nil, errOAuthSession
	}
	if cancel && (s.state == "waiting" || s.state == "exchanging") {
		s.cancel()
		s.state = "cancelled"
		s.authorizationURL = ""
	}
	return map[string]string{"session": s.id, "id": s.connection, "state": s.state}, nil
}

func (e *Engine) closeGoogleConsent() {
	e.consentMu.Lock()
	e.consentClosed = true
	s := e.consent
	if s != nil {
		s.cancel()
	}
	e.consentMu.Unlock()
	if s != nil {
		<-s.done
	}
}

// Discover the in-memory session after UI reload. The URL is returned only while
// awaiting consent; no verifier, code, secret or token is part of this view.
func (e *Engine) currentGoogleConsent(data []byte) (any, error) {
	var req struct{}
	if decode(data, &req) != nil {
		return nil, errOAuthSession
	}
	e.consentMu.Lock()
	defer e.consentMu.Unlock()
	if e.consent == nil || e.consentClosed {
		return map[string]string{"state": "none"}, nil
	}
	s := e.consent
	result := map[string]string{"session": s.id, "id": s.connection, "state": s.state}
	if s.state == "waiting" {
		result["authorization_url"] = s.authorizationURL
	}
	return result, nil
}
