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

func slackRequest(body, secret string, now time.Time) *http.Request {
	r := httptest.NewRequest("POST", "/hooks/slack", strings.NewReader(body))
	stamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v0:" + stamp + ":" + body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Slack-Request-Timestamp", stamp)
	r.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(mac.Sum(nil)))
	return r
}

func TestSlackOfficialSignatureVector(t *testing.T) {
	// Fixture from https://docs.slack.dev/authentication/verifying-requests-from-slack/
	body := `token=xyzz0WbapA4vBCDEFasx0q6G&team_id=T1DC2JH3J&team_domain=testteamnow&channel_id=G8PSS9T3V&channel_name=foobar&user_id=U2CERLKJA&user_name=roadrunner&command=%2Fwebhook-collect&text=&response_url=https%3A%2F%2Fhooks.slack.com%2Fcommands%2FT1DC2JH3J%2F397700885554%2F96rGlfmibIGlgcZRskXaIFfN&trigger_id=398738663015.47445629121.803a0bc887a14d10d2c447fce8b6703c`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("X-Slack-Request-Timestamp", "1531420618")
	r.Header.Set("X-Slack-Signature", "v0=a2114d57b48eac39b9ad189dd8316235a7b4a8d21a10bd27519666489c69b503")
	now := time.Unix(1531420618, 0)
	if _, err := authenticate(Entry{Auth: "slack"}, r, []byte(body), "8f742231b10e8888abcd99yyyzzz85a5", now); err != nil {
		t.Fatal(err)
	}
	if _, err := authenticate(Entry{Auth: "slack"}, r, []byte(body), "8f742231b10e8888abcd99yyyzzz85a5", now.Add(301*time.Second)); err == nil {
		t.Fatal("stale official fixture accepted")
	}
}

func TestSlackChallengeAndDurableEvents(t *testing.T) {
	e := testEngine(t)
	secret := "local-slack-fixture-secret"
	credential, _ := json.Marshal(map[string]string{"id": "slack-signing", "value": secret, "backend": "file"})
	if _, err := e.putSecret(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	c := notificationConfig()
	c.Entries = []Entry{{ID: "slack", Auth: "slack", Secret: "slack-signing", Format: "json", Enabled: true}}
	c.Flows[0].Source = "entry:slack"
	activateTest(t, e, c)
	handler := e.IngressHandler()
	now := time.Now()
	challenge := `{"type":"url_verification","challenge":"fixture-challenge"}`
	for _, valid := range []bool{false, true} {
		r := slackRequest(challenge, secret, now)
		if !valid {
			r.Header.Set("X-Slack-Signature", "v0=invalid")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if valid {
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"challenge":"fixture-challenge"`) {
				t.Fatal(w.Code, w.Body.String())
			}
		} else if w.Code != 401 || strings.Contains(w.Body.String(), "fixture-challenge") {
			t.Fatal("unauthenticated challenge leaked", w.Code)
		}
	}
	var count int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 0 {
		t.Fatal("challenge queued automation")
	}
	malformed := `{"type":"url_verification","challenge":123}`
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, slackRequest(malformed, secret, now))
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	body := `{"type":"event_callback","event_id":"Ev1","event":{"type":"app_mention","text":"fixture"}}`
	for i := 0; i < 2; i++ {
		r := slackRequest(body, secret, now.Add(time.Duration(i)*time.Second))
		r.Header.Set("X-Slack-Retry-Num", strconv.Itoa(i))
		r.Header.Set("X-Quatrro-Delivery", strconv.Itoa(i))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	e.db.QueryRow("SELECT count(*) FROM executions").Scan(&count)
	if count != 1 {
		t.Fatal("retry duplicated effects", count)
	}
	// Force a storage failure: even authenticated events must not receive success.
	if _, err := e.db.Exec(`CREATE TRIGGER reject_provider BEFORE INSERT ON events BEGIN SELECT RAISE(ABORT,'fixture full'); END`); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, slackRequest(strings.Replace(body, "Ev1", "Ev2", 1), secret, now))
	if w.Code != 503 {
		t.Fatal("acknowledged failed commit", w.Code)
	}
}

func TestProviderTamperingAmbiguityAndFormats(t *testing.T) {
	now := time.Now()
	body := `{"type":"event_callback"}`
	secret := "slack-fixture-secret"
	cases := []func(*http.Request){
		func(r *http.Request) {
			r.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(now.Add(-301*time.Second).Unix(), 10))
		},
		func(r *http.Request) {
			r.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(now.Add(301*time.Second).Unix(), 10))
		},
		func(r *http.Request) { r.Header.Set("X-Slack-Signature", "v1="+strings.Repeat("0", 64)) },
		func(r *http.Request) { r.Header.Add("X-Slack-Signature", r.Header.Get("X-Slack-Signature")) },
		func(r *http.Request) { r.Header.Del("X-Slack-Request-Timestamp") },
	}
	for i, mutate := range cases {
		r := slackRequest(body, secret, now)
		mutate(r)
		if _, err := authenticate(Entry{Auth: "slack"}, r, []byte(body), secret, now); err == nil {
			t.Fatal("accepted tamper case", i)
		}
	}
	r := slackRequest(body, secret, now)
	if _, err := authenticate(Entry{Auth: "slack"}, r, []byte(body+" "), secret, now); err == nil {
		t.Fatal("body tampering accepted")
	}
	if _, err := authenticate(Entry{Auth: "unknown"}, r, []byte(body), secret, now); err == nil {
		t.Fatal("unknown provider accepted")
	}
	c := EmptyConfig()
	c.Entries = []Entry{{ID: "slack", Auth: "slack", Secret: "signing", Format: "xml"}}
	if err := c.Validate(); err == nil {
		t.Fatal("incompatible Slack format accepted")
	}
	c.Entries[0].Format = "form"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
