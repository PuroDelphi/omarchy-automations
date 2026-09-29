package core

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
)

func TestWorkerPauseRevocationAndSequentialSteps(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	c.Flows[0].Steps = []string{"notice", "notice"}
	activateTest(t, e, c)
	var calls atomic.Int32
	e.actionRunner = func(context.Context, Action, Event) error { calls.Add(1); return nil }
	if _, err := e.ingest(ctx, Event{Source: "local:test", Type: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.control(ctx, []byte(`{"paused":true,"admission":"retain"}`)); err != nil {
		t.Fatal(err)
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("pause ignored")
	}
	if _, err := e.control(ctx, []byte(`{"paused":false,"admission":"retain"}`)); err != nil {
		t.Fatal(err)
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("did not dispatch first step")
	}
	if _, err := e.revoke(ctx, []byte(`{"scope":"flow:deploy:action:notice"}`)); err != nil {
		t.Fatal(err)
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("revoked action executed")
	}
	var state string
	e.db.QueryRow("SELECT state FROM executions").Scan(&state)
	if state != "denied" {
		t.Fatal(state)
	}
	var completed string
	if err := e.db.QueryRow("SELECT state FROM steps WHERE step=0").Scan(&completed); err != nil || completed != "completed" {
		t.Fatal("successful step mislabeled", completed, err)
	}
	var message string
	if err := e.db.QueryRow("SELECT state,message FROM steps WHERE step=1").Scan(&state, &message); err != nil || state != "denied" || message != "action permission missing or changed" {
		t.Fatal("revocation missing from step history", state, message, err)
	}
	if err := e.tick(ctx); err != nil || calls.Load() != 1 {
		t.Fatal("denied step retried", err, calls.Load())
	}
}
func TestJSONMappingCannotInjectFields(t *testing.T) {
	b, err := renderJSON(`{"message":"{{data.text}}"}`, Event{Data: map[string]any{"text": `x","admin":true,"x":"`}})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err = json.Unmarshal(b, &result); err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal("injected keys", string(b))
	}
}

func TestJSONMappingPreservesScalarTypes(t *testing.T) {
	b, err := renderJSON(`{"count":"{{data.count}}","active":"{{data.active}}"}`, Event{Data: map[string]any{"count": float64(7), "active": true}})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err = json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["count"] != float64(7) || got["active"] != true {
		t.Fatal("lost types", string(b))
	}
}
