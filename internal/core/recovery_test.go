package core

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestExplicitCancelDoesNotRestartCommand(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	e.ingest(ctx, Event{Source: "local:test"})
	started := make(chan struct{})
	e.actionRunner = func(ctx context.Context, _ Action, _ Event) error { close(started); <-ctx.Done(); return ctx.Err() }
	done := make(chan error, 1)
	go func() { done <- e.tick(ctx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("action not started")
	}
	if _, err := e.cancelJobs(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var state string
	e.db.QueryRow("SELECT state FROM executions").Scan(&state)
	if state != "uncertain" {
		t.Fatal("cancelled effect automatically scheduled", state)
	}
}

func TestHTTPRetryPersistsRequestAcrossRestart(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	c.Destinations = []Destination{{ID: "out", URL: "https://example.test", Method: "POST"}}
	c.Actions = []Action{{ID: "notice", Kind: "http", Destination: "out", Body: `{"message":"{{data.message}}"}`}}
	activateTest(t, e, c)
	e.ingest(ctx, Event{Source: "local:test", Data: map[string]any{"message": "original"}})
	var key, body string
	e.deliverySender = func(_ context.Context, _ Destination, k string, b []byte) deliveryResult {
		key = k
		body = string(b)
		return deliveryResult{Status: 429, Retry: true, After: time.Minute}
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	var attempts, next int
	e.db.QueryRow("SELECT attempts,next_at FROM outbox").Scan(&attempts, &next)
	if attempts != 1 || next < int(time.Now().Add(59*time.Second).Unix()) {
		t.Fatal("retry timing not persisted", attempts, next)
	}
	p := e.paths
	e.Close()
	other, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	called := false
	other.deliverySender = func(_ context.Context, _ Destination, k string, b []byte) deliveryResult {
		called = true
		if key != k || body != string(b) {
			t.Error("request identity changed")
		}
		return deliveryResult{Status: 204}
	}
	if err = other.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("ignored persisted retry delay")
	}
	other.db.Exec("UPDATE outbox SET next_at=0")
	if err = other.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("retry did not occur")
	}
	if err = other.tick(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	other.db.QueryRow("SELECT state FROM executions").Scan(&state)
	if state != "completed" {
		t.Fatal(state)
	}
	var id string
	other.db.QueryRow("SELECT id FROM executions").Scan(&id)
	request, _ := json.Marshal(map[string]string{"id": id})
	if _, err = other.retry(ctx, request); err == nil {
		t.Fatal("allowed retry of completed delivery")
	}
}

func TestInterruptedCommandRecoveryDoesNotRepeatEffect(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	e.ingest(ctx, Event{Source: "local:test"})
	e.db.Exec("UPDATE executions SET state='running'")
	run, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	e.actionRunner = func(context.Context, Action, Event) error {
		t.Error("repeated uncertain action")
		return errors.New("should not execute")
	}
	go func() { done <- e.Run(run) }()
	deadline := time.Now().Add(time.Second)
	for {
		var state string
		e.db.QueryRow("SELECT state FROM executions").Scan(&state)
		if state == "uncertain" {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("recovery not applied")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestCancelledHTTPIsNotRescheduled(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	c.Destinations = []Destination{{ID: "out", URL: "https://example.test", Method: "POST"}}
	c.Actions = []Action{{ID: "notice", Kind: "http", Destination: "out", Body: `{"ok":true}`}}
	activateTest(t, e, c)
	e.ingest(ctx, Event{Source: "local:test"})
	started := make(chan struct{})
	calls := 0
	e.deliverySender = func(ctx context.Context, _ Destination, _ string, _ []byte) deliveryResult {
		calls++
		close(started)
		<-ctx.Done()
		return deliveryResult{Retry: true, Message: "cancelled transport"}
	}
	done := make(chan error, 1)
	go func() { done <- e.tick(ctx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("delivery not started")
	}
	if _, err := e.cancelJobs(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	e.db.QueryRow("SELECT state FROM executions").Scan(&state)
	if state != "uncertain" || calls != 1 {
		t.Fatal("cancelled delivery retried", state, calls)
	}
	e.db.QueryRow("SELECT state FROM outbox").Scan(&state)
	if state != "uncertain" {
		t.Fatal("outbox state", state)
	}
}

func TestPersistedHTTPCompletionSurvivesInterruptedStepCommit(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	c.Destinations = []Destination{{ID: "out", URL: "https://example.test", Method: "POST"}}
	c.Actions = []Action{{ID: "notice", Kind: "http", Destination: "out", Body: `{}`}}
	activateTest(t, e, c)
	e.ingest(ctx, Event{Source: "local:test"})
	var id string
	e.db.QueryRow("SELECT id FROM executions").Scan(&id)
	e.db.Exec("UPDATE executions SET state='running'")
	e.db.Exec("INSERT INTO outbox(execution,step,body,attempts,next_at,created_at,state,last_status) VALUES(?,0,'{}',1,0,?,'delivered',204)", id, time.Now().Unix())
	e.deliverySender = func(context.Context, Destination, string, []byte) deliveryResult {
		t.Error("resent a committed delivery")
		return deliveryResult{Status: 204}
	}
	run, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- e.Run(run) }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var state string
		e.db.QueryRow("SELECT state FROM executions").Scan(&state)
		if state == "completed" {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("recovery failed", state)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestTerminalOutboxSurvivesFailedStepCommitAndRestart(t *testing.T) {
	for _, terminal := range []string{"delivered", "failed", "uncertain"} {
		t.Run(terminal, func(t *testing.T) {
			ctx := context.Background()
			e := testEngine(t)
			c := notificationConfig()
			c.Destinations = []Destination{{ID: "out", URL: "https://example.test", Method: "POST"}}
			c.Actions = []Action{{ID: "notice", Kind: "http", Destination: "out", Body: `{}`}}
			activateTest(t, e, c)
			if _, err := e.ingest(ctx, Event{Source: "local:test"}); err != nil {
				t.Fatal(err)
			}
			if _, err := e.db.Exec(`CREATE TRIGGER fail_step BEFORE INSERT ON steps BEGIN SELECT RAISE(ABORT,'step storage unavailable'); END`); err != nil {
				t.Fatal(err)
			}
			calls := 0
			e.deliverySender = func(context.Context, Destination, string, []byte) deliveryResult {
				calls++
				switch terminal {
				case "delivered":
					return deliveryResult{Status: 204}
				case "uncertain":
					if _, err := e.cancelJobs(ctx); err != nil {
						t.Fatal(err)
					}
					return deliveryResult{Retry: true}
				default:
					return deliveryResult{Status: 400}
				}
			}
			if err := e.tick(ctx); err == nil {
				t.Fatal("expected step persistence failure")
			}
			var state string
			if err := e.db.QueryRow("SELECT state FROM outbox").Scan(&state); err != nil || state != terminal || calls != 1 {
				t.Fatal("terminal result not persisted", state, calls, err)
			}
			if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil || state != "running" {
				t.Fatal("unexpected interrupted state", state, err)
			}
			if _, err := e.db.Exec("DROP TRIGGER fail_step"); err != nil {
				t.Fatal(err)
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
			reopened.deliverySender = func(context.Context, Destination, string, []byte) deliveryResult {
				t.Error("repeated persisted terminal delivery")
				return deliveryResult{Status: 204}
			}
			run, cancel := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- reopened.Run(run) }()
			defer func() {
				cancel()
				if err := <-done; err != nil {
					t.Error(err)
				}
			}()
			want := terminal
			if terminal == "delivered" {
				want = "completed"
			}
			deadline := time.Now().Add(3 * time.Second)
			for {
				if err := reopened.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil {
					t.Fatal(err)
				}
				if state == want {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("recovery state %s, want %s", state, want)
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}
