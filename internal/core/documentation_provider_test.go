package core

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDocumentedGitHubRelease(t *testing.T) {
	raw, err := os.ReadFile("../../examples/use-cases/github-release.json")
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err = decode(raw, &c); err != nil {
		t.Fatal(err)
	}
	if err = c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Entries[0].Enabled || c.Flows[0].Enabled {
		t.Fatal("example enabled")
	}
	c.Entries[0].Enabled = true
	c.Flows[0].Enabled = true
	e := testEngine(t)
	ctx := context.Background()
	secret := "public-test-only-github-signing-key"
	raw, _ = json.Marshal(map[string]string{"id": "github-signing", "value": secret, "backend": "file"})
	if _, err = e.putSecret(ctx, raw); err != nil {
		t.Fatal(err)
	}
	activateTest(t, e, c)
	for _, tag := range []string{"v1.2.0", "v1.2.1"} {
		payload, _ := json.Marshal(Event{Source: "entry:github-releases", Type: "webhook", Data: map[string]any{
			"action": "published", "repository": map[string]any{"full_name": "example/project"}, "release": map[string]any{"tag_name": tag},
		}})
		result, err := e.simulate(ctx, payload)
		if err != nil {
			t.Fatal(err)
		}
		matches := result.(map[string]any)["matches"].([]map[string]any)
		if len(matches) != 1 || matches[0]["steps"].([]map[string]any)[0]["body"] != "example/project: "+tag {
			t.Fatal(result)
		}
	}
	handler := e.IngressHandler()
	body := `{"action":"published","repository":{"full_name":"example/project"},"release":{"tag_name":"v1.2.0"}}`
	send := func(payload, signed, delivery string) int {
		r := httptest.NewRequest("POST", "/hooks/github-releases", strings.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-GitHub-Delivery", delivery)
		r.Header.Set("X-GitHub-Event", "not-trusted-for-filtering")
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(signed))
		r.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}
	if status := send(body, body, "release-1"); status != 202 {
		t.Fatal(status)
	}
	if status := send(body, body, "changed-delivery-header"); status != 202 {
		t.Fatal(status)
	}
	tampered := strings.Replace(body, "v1.2.0", "v1.2.1", 1)
	if status := send(tampered, body, "tampered"); status != 401 {
		t.Fatal(status)
	}
	other := strings.Replace(body, "example/project", "example/other", 1)
	if status := send(other, other, "other-repo"); status != 202 {
		t.Fatal(status)
	}
	draft := strings.Replace(body, "published", "created", 1)
	if status := send(draft, draft, "draft-release"); status != 202 {
		t.Fatal(status)
	}
	var events, jobs int
	if err = e.db.QueryRow("SELECT count(*) FROM events").Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err = e.db.QueryRow("SELECT count(*) FROM executions").Scan(&jobs); err != nil || events != 3 || jobs != 1 {
		t.Fatal(events, jobs, err)
	}
	var notice string
	e.actionRunner = func(_ context.Context, a Action, ev Event) error {
		var err error
		notice, err = render(a.Body, ev)
		return err
	}
	if err = e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if notice != "example/project: v1.2.0" {
		t.Fatal(notice)
	}
}
