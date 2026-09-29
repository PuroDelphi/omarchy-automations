package core

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDocumentedHTTPRecovery(t *testing.T) {
	raw, err := os.ReadFile("../../examples/use-cases/http-recovery.json")
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
	if c.Flows[0].Enabled {
		t.Fatal("example enabled")
	}
	c.Flows[0].Enabled = true
	e := testEngine(t)
	ctx := context.Background()
	raw, _ = json.Marshal(map[string]string{"id": "receiver-token", "value": "public-test-only-receiver-token", "backend": "file"})
	if _, err = e.putSecret(ctx, raw); err != nil {
		t.Fatal(err)
	}
	activateTest(t, e, c)
	for _, message := range []string{"Build finished", "Backup verified"} {
		raw, _ = json.Marshal(Event{Source: "local:delivery", Data: map[string]any{"message": message, "request": "demo-1"}})
		result, err := e.simulate(ctx, raw)
		if err != nil {
			t.Fatal(err)
		}
		steps := result.(map[string]any)["matches"].([]map[string]any)[0]["steps"].([]map[string]any)
		want, _ := json.Marshal(map[string]string{"message": message, "request": "demo-1"})
		if steps[0]["body"] != string(want) {
			t.Fatal(result)
		}
	}
	calls := 0
	var key, body string
	e.deliverySender = func(_ context.Context, _ Destination, k string, b []byte) deliveryResult {
		calls++
		if calls == 1 {
			key, body = k, string(b)
		} else if k != key || string(b) != body {
			t.Fatal("retry changed request identity")
		}
		switch calls {
		case 1:
			return deliveryResult{Status: 503, Retry: true, After: time.Minute}
		case 2:
			return deliveryResult{Status: 400}
		default:
			return deliveryResult{Status: 204}
		}
	}
	if _, err = e.ingest(ctx, Event{Source: "local:delivery", Data: map[string]any{"message": "Build finished", "request": "demo-1"}}); err != nil {
		t.Fatal(err)
	}
	tick := func() {
		t.Helper()
		if err := e.tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	state := func() string {
		t.Helper()
		var state string
		if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil {
			t.Fatal(err)
		}
		return state
	}
	tick()
	if state() != "pending" || calls != 1 {
		t.Fatal(state(), calls)
	}
	tick()
	if calls != 1 {
		t.Fatal("ignored retry delay")
	}
	// Advance this test's persisted deadline; production scheduling is unchanged.
	if _, err = e.db.Exec("UPDATE outbox SET next_at=0"); err != nil {
		t.Fatal(err)
	}
	tick()
	if state() != "failed" || calls != 2 {
		t.Fatal(state(), calls)
	}
	// Draft edits must not replace an already persisted request.
	c.Actions[0].Body = `{"message":"changed draft"}`
	raw, _ = json.Marshal(c)
	if _, err = e.saveDraft(ctx, raw); err != nil {
		t.Fatal(err)
	}
	var id string
	if err = e.db.QueryRow("SELECT id FROM executions").Scan(&id); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(map[string]string{"id": id})
	if _, err = e.control(ctx, []byte(`{"paused":true,"admission":"retain"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = e.retry(ctx, raw); err != nil {
		t.Fatal(err)
	}
	tick()
	if calls != 2 || state() != "pending" {
		t.Fatal("manual retry bypassed pause")
	}
	if _, err = e.control(ctx, []byte(`{"paused":false,"admission":"retain"}`)); err != nil {
		t.Fatal(err)
	}
	tick()
	tick()
	if calls != 3 || state() != "completed" {
		t.Fatal(calls, state())
	}
	if _, err = e.retry(ctx, raw); err == nil {
		t.Fatal("completed delivery retried")
	}
	if key == "" || body != `{"message":"Build finished","request":"demo-1"}` {
		t.Fatal(key, body)
	}
}
