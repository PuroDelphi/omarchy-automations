package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// inboundAdapter is the built-in provider contract. Verify runs on bounded raw
// bytes before decoding. Challenge only sees authenticated, parsed data and
// cannot perform network/OS effects. Registration is compile-time only.
type inboundAdapter struct {
	Version              int
	Verify               func(*http.Request, []byte, string, time.Time) (string, error)
	Challenge            func(map[string]any) (any, bool, error)
	Formats              map[string]bool // nil permits all supported codecs
	EmptyAcknowledgement bool
	SuccessStatus        int // normal events are acknowledged only after durable commit
}

func inboundProvider(name string) (inboundAdapter, bool) {
	adapter := inboundAdapter{Version: 1, SuccessStatus: http.StatusAccepted}
	switch name {
	case "hmac":
		adapter.Verify = verifyGenericHMAC
	case "bearer":
		adapter.Verify = verifyBearer
	case "github":
		adapter.Verify = verifyGitHub
	case "slack":
		adapter.Verify = verifySlack
		adapter.Challenge = slackChallenge
		adapter.Formats = map[string]bool{"json": true, "form": true}
		adapter.SuccessStatus = http.StatusOK
		adapter.EmptyAcknowledgement = true
	default:
		return inboundAdapter{}, false
	}
	return adapter, true
}

func authenticate(entry Entry, r *http.Request, raw []byte, secret string, now time.Time) (string, error) {
	adapter, ok := inboundProvider(entry.Auth)
	if !ok || adapter.Version != 1 || secret == "" {
		return "", errors.New("unsupported authentication or missing secret")
	}
	// Reject ambiguity instead of selecting one of several authentication headers.
	for _, key := range []string{"Authorization", "X-Quatrro-Timestamp", "X-Quatrro-Delivery", "X-Quatrro-Signature", "X-Hub-Signature-256", "X-GitHub-Delivery", "X-Slack-Request-Timestamp", "X-Slack-Signature"} {
		if len(r.Header.Values(key)) > 1 {
			return "", errors.New("duplicate authentication header")
		}
	}
	return adapter.Verify(r, raw, secret, now)
}

func checkTimestamp(stamp string, now time.Time) error {
	sec, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || strconv.FormatInt(sec, 10) != stamp {
		return errors.New("invalid timestamp")
	}
	delta := now.Sub(time.Unix(sec, 0))
	if delta > 5*time.Minute || delta < -5*time.Minute {
		return errors.New("expired timestamp")
	}
	return nil
}

func checkSignature(header, prefix string, message []byte, secret string) error {
	if !strings.HasPrefix(header, prefix) {
		return errors.New("missing signature")
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil || len(provided) != sha256.Size {
		return errors.New("invalid signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return errors.New("signature mismatch")
	}
	return nil
}

func bodyIdentity(provider string, raw []byte) string {
	digest := sha256.Sum256(raw)
	return provider + ":" + hex.EncodeToString(digest[:])
}

func verifyBearer(r *http.Request, _ []byte, secret string, _ time.Time) (string, error) {
	if !hmac.Equal([]byte(r.Header.Get("Authorization")), []byte("Bearer "+secret)) {
		return "", errors.New("token mismatch")
	}
	return r.Header.Get("X-Quatrro-Delivery"), nil
}

func verifyGenericHMAC(r *http.Request, raw []byte, secret string, now time.Time) (string, error) {
	stamp, delivery := r.Header.Get("X-Quatrro-Timestamp"), r.Header.Get("X-Quatrro-Delivery")
	if err := checkTimestamp(stamp, now); err != nil {
		return "", err
	}
	if delivery == "" {
		return "", errors.New("missing delivery ID")
	}
	err := checkSignature(r.Header.Get("X-Quatrro-Signature"), "sha256=", append([]byte(stamp+"."+delivery+"."), raw...), secret)
	return delivery, err
}

func verifyGitHub(r *http.Request, raw []byte, secret string, _ time.Time) (string, error) {
	if r.Header.Get("X-GitHub-Delivery") == "" {
		return "", errors.New("missing delivery ID")
	}
	if err := checkSignature(r.Header.Get("X-Hub-Signature-256"), "sha256=", raw, secret); err != nil {
		return "", err
	}
	// GitHub does not sign delivery/event headers. Identity must not depend on them.
	return bodyIdentity("github", raw), nil
}

func verifySlack(r *http.Request, raw []byte, secret string, now time.Time) (string, error) {
	stamp := r.Header.Get("X-Slack-Request-Timestamp")
	if err := checkTimestamp(stamp, now); err != nil {
		return "", err
	}
	if err := checkSignature(r.Header.Get("X-Slack-Signature"), "v0=", append([]byte("v0:"+stamp+":"), raw...), secret); err != nil {
		return "", err
	}
	// A provider retry can have a new timestamp; identical authenticated bodies
	// share identity across retries. Retry headers never grant additional effects.
	return bodyIdentity("slack", raw), nil
}

func slackChallenge(data map[string]any) (any, bool, error) {
	if data["type"] != "url_verification" {
		return nil, false, nil
	}
	value, ok := data["challenge"].(string)
	if !ok || value == "" || len(value) > 2048 {
		return nil, true, errors.New("invalid challenge")
	}
	return map[string]string{"challenge": value}, true, nil
}
