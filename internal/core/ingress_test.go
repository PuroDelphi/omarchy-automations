package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSignedWebhookPersistsOnceAndRejectsTampering(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	secret := "a-test-secret-with-enough-entropy"
	raw, _ := json.Marshal(map[string]string{"id": "incoming", "value": secret, "backend": "file"})
	if _, err := e.putSecret(ctx, raw); err != nil {
		t.Fatal(err)
	}
	c := notificationConfig()
	c.Entries = []Entry{{ID: "deploy", Auth: "hmac", Secret: "incoming", Format: "json", Enabled: true}}
	c.Flows[0].Source = "entry:deploy"
	activateTest(t, e, c)
	handler := e.IngressHandler()
	body := `{"state":"failed"}`
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	request := func(body, signed, stamp string) *http.Request {
		r := httptest.NewRequest("POST", "/hooks/deploy", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Quatrro-Event", "unsigned-event-must-not-control-rules")
		r.Header.Set("X-Quatrro-Timestamp", stamp)
		r.Header.Set("X-Quatrro-Delivery", "delivery-1")
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stamp + ".delivery-1." + signed))
		r.Header.Set("X-Quatrro-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		return r
	}
	for range 2 {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request(body, body, stamp))
		if w.Code != 202 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var n int
	e.db.QueryRow("SELECT count(*) FROM executions").Scan(&n)
	if n != 1 {
		t.Fatal("deduplication failed", n)
	}
	var stored string
	e.db.QueryRow("SELECT payload FROM events LIMIT 1").Scan(&stored)
	var ev Event
	json.Unmarshal([]byte(stored), &ev)
	if ev.Type != "webhook" {
		t.Fatal("unsigned event header influenced normalized type")
	}
	for _, r := range []*http.Request{request(`{"state":"ok"}`, body, stamp), request(body, body, strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10))} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatal("invalid delivery accepted", w.Code)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request(strings.Repeat("x", 262145), body, stamp))
	if w.Code != 413 {
		t.Fatal("large body accepted", w.Code)
	}
}

func TestGitHubOfficialSignatureVector(t *testing.T) {
	// https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-GitHub-Delivery", "example")
	r.Header.Set("X-Hub-Signature-256", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
	if _, err := authenticate(Entry{Auth: "github"}, r, []byte("Hello, World!"), "It's a Secret to Everybody", time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestGitHubReplayCannotChangeDeliveryHeader(t *testing.T) {
	body := []byte(`{"action":"completed"}`)
	secret := "public-test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	r := httptest.NewRequest("POST", "/", nil)
	r.Header.Set("X-GitHub-Delivery", "first")
	r.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	first, err := authenticate(Entry{Auth: "github"}, r, body, secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("X-GitHub-Delivery", "modified")
	second, err := authenticate(Entry{Auth: "github"}, r, body, secret, time.Now())
	if err != nil || first != second {
		t.Fatal("unsigned header changed replay identity")
	}
}
