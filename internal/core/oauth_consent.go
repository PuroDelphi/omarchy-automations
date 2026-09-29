package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var errOAuthCallback = errors.New("OAuth callback invalid, expired or already consumed")
var errOAuthConsentDenied = errors.New("OAuth consent was not granted")

// Private, short-lived session. Never serialize verifier or client secret into
// browser URLs, public status, configuration, logs or persistence.
type googleAuthorization struct {
	requestedScopes                                   []string
	codeHash                                          [32]byte
	exchanged                                         bool
	mu                                                sync.Mutex
	clientID, clientSecret, redirect, verifier, state string
	expires                                           time.Time
	consumed                                          bool
}

func googleLoopbackRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Path != "/oauth/callback" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	return err == nil && port > 0 && port <= 65535
}

func newGoogleAuthorization(clientID, clientSecret, redirect string, scopes []string, now time.Time) (*googleAuthorization, string, error) {
	if !oauthValue(clientID) || !oauthValue(clientSecret) || !googleLoopbackRedirect(redirect) || len(scopes) == 0 || len(scopes) > 16 {
		return nil, "", errOAuthCredentials
	}
	seen := map[string]bool{}
	for _, scope := range scopes {
		u, err := url.Parse(scope)
		if err != nil || len(scope) > 256 || !oauthValue(scope) || u.Scheme != "https" || u.Host != "www.googleapis.com" || u.User != nil || !strings.HasPrefix(u.Path, "/auth/") || len(u.Path) <= 6 || u.RawQuery != "" || u.Fragment != "" || seen[scope] {
			return nil, "", errOAuthCredentials
		}
		seen[scope] = true
	}
	randomValue := func() string { var b [32]byte; rand.Read(b[:]); return base64.RawURLEncoding.EncodeToString(b[:]) }
	a := &googleAuthorization{requestedScopes: append([]string(nil), scopes...), clientID: clientID, clientSecret: clientSecret, redirect: redirect, verifier: randomValue(), state: randomValue(), expires: now.Add(10 * time.Minute)}
	hash := sha256.Sum256([]byte(a.verifier))
	params := url.Values{"client_id": {clientID}, "redirect_uri": {redirect}, "response_type": {"code"}, "scope": {strings.Join(scopes, " ")}, "state": {a.state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(hash[:])}, "code_challenge_method": {"S256"}, "access_type": {"offline"}, "prompt": {"consent"}}
	return a, "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode(), nil
}

// Callback validation is separate from the listener and exchange, so malformed
// requests cannot consume the session or trigger provider traffic.
func (a *googleAuthorization) consumeCallback(r *http.Request, now time.Time) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	redirect, _ := url.Parse(a.redirect)
	if a.consumed || !now.Before(a.expires) || r.Method != "GET" || r.Host != redirect.Host || r.URL.Path != redirect.Path || len(r.URL.RawQuery) > 16384 {
		return "", errOAuthCallback
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q["state"]) != 1 || subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(a.state)) != 1 {
		return "", errOAuthCallback
	}
	if len(q["error"]) == 1 && q.Get("error") != "" && len(q["code"]) == 0 {
		a.consumed = true
		return "", errOAuthConsentDenied
	}
	if len(q["error"]) != 0 || len(q["code"]) != 1 || !oauthValue(q.Get("code")) {
		return "", errOAuthCallback
	}
	a.consumed = true
	a.codeHash = sha256.Sum256([]byte(q.Get("code")))
	return q.Get("code"), nil
}

func (a *googleAuthorization) exchange(ctx context.Context, client *http.Client, code string, now time.Time) (oauthToken, error) {
	a.mu.Lock()
	valid := oauthValue(code) && a.consumed && !a.exchanged && now.Before(a.expires) && a.codeHash == sha256.Sum256([]byte(code))
	if valid {
		a.exchanged = true
	}
	a.mu.Unlock()
	if !valid {
		return oauthToken{}, errOAuthCallback
	}
	body := url.Values{"client_id": {a.clientID}, "client_secret": {a.clientSecret}, "code": {code}, "code_verifier": {a.verifier}, "redirect_uri": {a.redirect}, "grant_type": {"authorization_code"}}
	// No fallback refresh token for initial consent: offline authorization must
	// return one before this connection can be persisted.
	token, err := googleTokenRequest(ctx, client, body, "", now)
	if err != nil {
		return oauthToken{}, err
	}
	if !oauthScopesInclude(token.Scope, a.requestedScopes) {
		return oauthToken{}, errOAuthScopes
	}
	return token, nil
}
