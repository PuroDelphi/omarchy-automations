package core

import (
	"context"
	"testing"
	"time"
)

func TestHTTPStorageBoundariesPreventUntrackedDispatch(t *testing.T) {
	for _, boundary := range []string{"outbox", "claim", "attempt", "result"} {
		t.Run(boundary, func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			c := notificationConfig()
			c.Destinations = []Destination{{ID: "out", URL: "https://example.test", Method: "POST"}}
			c.Actions = []Action{{ID: "notice", Kind: "http", Destination: "out", Body: `{}`}}
			activateTest(t, e, c)
			if _, err := e.ingest(ctx, Event{Source: "local:test"}); err != nil {
				t.Fatal(err)
			}
			triggers := map[string]string{
				"outbox":  "BEFORE INSERT ON outbox",
				"claim":   "BEFORE UPDATE ON executions WHEN NEW.state='running'",
				"attempt": "BEFORE UPDATE ON outbox WHEN NEW.attempts>OLD.attempts",
				"result":  "BEFORE UPDATE ON outbox WHEN NEW.state='delivered'",
			}
			if _, err := e.db.Exec("CREATE TRIGGER storage_fault " + triggers[boundary] + " BEGIN SELECT RAISE(ABORT,'fixture storage failure'); END"); err != nil {
				t.Fatal(err)
			}
			calls := 0
			e.deliverySender = func(context.Context, Destination, string, []byte) deliveryResult {
				calls++
				return deliveryResult{Status: 204}
			}
			if err := e.tick(ctx); err == nil {
				t.Fatal("expected storage failure")
			}
			want := 0
			if boundary == "result" {
				want = 1
			}
			if calls != want {
				t.Fatal("effect crossed uncommitted boundary", calls, want)
			}
			if _, err := e.db.Exec("DROP TRIGGER storage_fault"); err != nil {
				t.Fatal(err)
			}
			if boundary == "attempt" || boundary == "result" {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
				if calls != want {
					t.Fatal("retried interrupted effect without recovery")
				}
			} else {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
				if calls != 1 {
					t.Fatal("failed to resume safe pending work")
				}
			}
		})
	}
}

func TestFailedCancellationDoesNotAcknowledgeOrSignal(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	if _, err := e.ingest(ctx, Event{Source: "local:test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(`CREATE TRIGGER fail_cancel BEFORE UPDATE ON executions WHEN NEW.state='cancelled' BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	signalled := false
	e.running["fixture"] = func() { signalled = true }
	if _, err := e.cancelJobs(ctx); err == nil {
		t.Fatal("acknowledged failed cancellation")
	}
	if signalled {
		t.Fatal("signalled before durable cancellation")
	}
	var state string
	if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil || state != "pending" {
		t.Fatal(state, err)
	}
}

func TestLocalStorageBoundariesKeepEffectsUncertain(t *testing.T) {
	for _, boundary := range []string{"claim", "step", "advance"} {
		t.Run(boundary, func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			activateTest(t, e, notificationConfig())
			if _, err := e.ingest(ctx, Event{Source: "local:test"}); err != nil {
				t.Fatal(err)
			}
			triggers := map[string]string{
				"claim":   "BEFORE UPDATE ON executions WHEN NEW.state='running'",
				"step":    "BEFORE INSERT ON steps",
				"advance": "BEFORE UPDATE ON executions WHEN NEW.step>OLD.step",
			}
			if _, err := e.db.Exec("CREATE TRIGGER local_fault " + triggers[boundary] + " BEGIN SELECT RAISE(ABORT,'fixture'); END"); err != nil {
				t.Fatal(err)
			}
			calls := 0
			e.actionRunner = func(context.Context, Action, Event) error { calls++; return nil }
			if err := e.tick(ctx); err == nil {
				t.Fatal("storage failure not reported")
			}
			want := 1
			if boundary == "claim" {
				want = 0
			}
			if calls != want {
				t.Fatal("effect crossed claim boundary", calls, want)
			}
			var steps, step int
			var state string
			if err := e.db.QueryRow("SELECT count(*) FROM steps").Scan(&steps); err != nil || steps != 0 {
				t.Fatal("partial step commit", steps, err)
			}
			if err := e.db.QueryRow("SELECT state,step FROM executions").Scan(&state, &step); err != nil || step != 0 {
				t.Fatal(state, step, err)
			}
			expected := "running"
			if boundary == "claim" {
				expected = "pending"
			}
			if state != expected {
				t.Fatal(state, expected)
			}
			if _, err := e.db.Exec("DROP TRIGGER local_fault"); err != nil {
				t.Fatal(err)
			}
			if err := e.tick(ctx); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("effect repeated after storage recovered", calls)
			}
			paths := e.paths
			e.Close()
			reopened, err := Open(paths)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			reopened.actionRunner = func(context.Context, Action, Event) error { t.Error("recovery repeated local effect"); return nil }
			run, cancel := context.WithTimeout(ctx, 2*time.Second)
			done := make(chan error, 1)
			go func() { done <- reopened.Run(run) }()
			defer func() {
				cancel()
				if err := <-done; err != nil {
					t.Error(err)
				}
			}()
			expected = "uncertain"
			if boundary == "claim" {
				expected = "completed"
			}
			for {
				if err := reopened.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil {
					t.Fatal(err)
				}
				if state == expected {
					break
				}
				if run.Err() != nil {
					t.Fatalf("state %s, wanted %s", state, expected)
				}
				time.Sleep(time.Millisecond)
			}
			var integrity string
			if err := reopened.db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
				t.Fatal(integrity, err)
			}
		})
	}
}
