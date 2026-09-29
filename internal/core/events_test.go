package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func activateTest(t *testing.T, e *Engine, c Config) {
	t.Helper()
	ctx := context.Background()
	raw, _ := json.Marshal(c)
	if _, err := e.saveDraft(ctx, raw); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(map[string]any{"hash": canonicalHash(c), "grants": capabilities(c)})
	if _, err := e.activate(ctx, raw); err != nil {
		t.Fatal(err)
	}
}
func TestPersistentInboxDeduplicatesAndPinsRevision(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	c.Flows[0].Conditions = []Condition{{Field: "data.state", Op: "eq", Value: "failed"}}
	activateTest(t, e, c)
	ev := Event{ID: "delivery-1", Source: "local:test", Type: "deploy", Data: map[string]any{"state": "failed"}}
	for range 2 {
		if _, err := e.ingest(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	e.db.QueryRow("SELECT count(*) FROM executions").Scan(&n)
	if n != 1 {
		t.Fatalf("executions=%d", n)
	}
	ev.ID = "delivery-2"
	ev.Data["state"] = "ok"
	if _, err := e.ingest(ctx, ev); err != nil {
		t.Fatal(err)
	}
	e.db.QueryRow("SELECT count(*) FROM executions").Scan(&n)
	if n != 1 {
		t.Fatal("condition ignored")
	}
	original := canonicalHash(c)
	c.Actions[0].Body = "New revision"
	activateTest(t, e, c)
	var revision string
	e.db.QueryRow("SELECT revision FROM executions LIMIT 1").Scan(&revision)
	if revision != original {
		t.Fatal("execution revision mutated")
	}
	p := e.paths
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopened.db.QueryRow("SELECT count(*) FROM executions").Scan(&n)
	if n != 1 {
		t.Fatal("lost execution after restart")
	}
}
func TestFileSecretNeverExportedAndRejectsUnsafeMode(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	value := "sixteen-plus-secret-characters"
	raw, _ := json.Marshal(map[string]string{"id": "demo", "backend": "file", "value": value})
	result, err := e.putSecret(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(result)
	if strings.Contains(string(b), value) {
		t.Fatal("value leaked")
	}
	got, err := e.secret(ctx, "demo")
	if err != nil || got != value {
		t.Fatal("secret roundtrip", err)
	}
	p := filepath.Join(e.paths.Config, "secrets", "demo")
	if err := os.Chmod(p, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.secret(ctx, "demo"); err == nil {
		t.Fatal("read insecure secret")
	}
}
func TestTemplateDoesNotEvaluateCode(t *testing.T) {
	ev := Event{Data: map[string]any{"message": "$(touch /tmp/never)"}}
	s, err := render("Result {{data.message}}", ev)
	if err != nil || s != "Result $(touch /tmp/never)" {
		t.Fatal(s, err)
	}
	if _, err := render("{{data.missing}}", ev); err == nil {
		t.Fatal("missing field accepted")
	}
}
