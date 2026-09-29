package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"quatrro.local/automations/internal/paths"
)

// A separate process dies without Close, deferred cleanup or SQLite checkpoint.
// The effect marker models an external side effect that cannot join the DB tx.
func TestCrashProcessFixture(t *testing.T) {
	phase := os.Getenv("QUATRRO_TEST_CRASH_PHASE")
	if phase == "" {
		return
	}
	p, err := paths.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}
	e, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c := notificationConfig()
	if phase == "http-after-effect" || phase == "http-cancel-ack" {
		c.Destinations = []Destination{{ID: "out", URL: "https://example.test", Method: "POST"}}
		c.Actions = []Action{{ID: "notice", Kind: "http", Destination: "out", Body: `{"fixed":true}`}}
	}
	activateTest(t, e, c)
	if _, err := e.ingest(ctx, Event{ID: "crash-event", Source: "local:test"}); err != nil {
		t.Fatal(err)
	}
	crash := func() {
		if err := syscall.Kill(os.Getpid(), syscall.SIGKILL); err != nil {
			t.Fatal(err)
		}
		select {} // SIGKILL must be observed; no graceful fallback.
	}
	mark := func(value string) {
		f, err := os.OpenFile(filepath.Join(p.State, "effect"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString(value); err != nil {
			t.Fatal(err)
		}
		if err := f.Sync(); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if phase == "before-dispatch" {
		crash()
	}
	e.actionRunner = func(context.Context, Action, Event) error {
		if phase == "after-claim" {
			crash()
		}
		mark("performed")
		if phase == "after-effect" {
			crash()
		}
		return nil
	}
	e.deliverySender = func(_ context.Context, _ Destination, key string, body []byte) deliveryResult {
		mark(key + "\n" + string(body))
		if phase == "http-cancel-ack" {
			if _, err := e.cancelJobs(ctx); err != nil {
				t.Fatal(err)
			}
		}
		crash()
		return deliveryResult{}
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	if phase != "after-step-commit" {
		t.Fatal("fixture phase did not crash")
	}
	crash()
}

func TestAbruptProcessCrashRecovery(t *testing.T) {
	for _, phase := range []string{"before-dispatch", "after-claim", "after-effect", "after-step-commit", "http-after-effect", "http-cancel-ack"} {
		t.Run(phase, func(t *testing.T) {
			profile := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCrashProcessFixture$")
			cmd.Env = append(os.Environ(), "QUATRRO_PROFILE="+profile, "QUATRRO_TEST_CRASH_PHASE="+phase)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || !exitErr.ProcessState.Sys().(syscall.WaitStatus).Signaled() || exitErr.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
				t.Fatalf("fixture did not die abruptly: %v %s", err, output)
			}
			p := paths.Paths{Config: filepath.Join(profile, "config"), State: filepath.Join(profile, "state"), Runtime: filepath.Join(profile, "runtime")}
			e, err := Open(p)
			if err != nil {
				t.Fatal(err)
			}
			defer e.Close()
			var integrity string
			if err := e.db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
				t.Fatal(integrity, err)
			}
			if phase == "http-after-effect" || phase == "http-cancel-ack" {
				var attempts int
				if err := e.db.QueryRow("SELECT attempts FROM outbox").Scan(&attempts); err != nil || attempts != 1 {
					t.Fatal("crash lost reserved attempt", attempts, err)
				}
			}
			marker, markerErr := os.ReadFile(filepath.Join(p.State, "effect"))
			if phase == "before-dispatch" || phase == "after-claim" {
				if !os.IsNotExist(markerErr) {
					t.Fatal("unexpected effect", markerErr)
				}
			} else if markerErr != nil {
				t.Fatal(markerErr)
			}
			e.actionRunner = func(context.Context, Action, Event) error {
				if phase != "before-dispatch" {
					t.Error("repeated an uncertain or committed effect")
				}
				return nil
			}
			e.deliverySender = func(_ context.Context, _ Destination, key string, body []byte) deliveryResult {
				if phase != "http-after-effect" || string(marker) != key+"\n"+string(body) {
					t.Error("HTTP retry changed identity or body")
				}
				return deliveryResult{Status: 204}
			}
			run, stop := context.WithCancel(ctx)
			done := make(chan error, 1)
			go func() { done <- e.Run(run) }()
			defer func() {
				stop()
				if err := <-done; err != nil {
					t.Error(err)
				}
			}()
			want := "completed"
			if phase == "after-claim" || phase == "after-effect" || phase == "http-cancel-ack" {
				want = "uncertain"
			}
			for {
				var state string
				if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil {
					t.Fatal(err)
				}
				if state == want {
					break
				}
				if ctx.Err() != nil {
					t.Fatalf("recovery remained %s, want %s", state, want)
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func TestNotificationWithoutSessionFailsWithoutRetry(t *testing.T) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		t.Skip("notify-send not installed")
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(t.TempDir(), "absent-bus"))
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	if _, err := e.ingest(ctx, Event{Source: "local:test"}); err != nil {
		t.Fatal(err)
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	var state, message string
	if err := e.db.QueryRow("SELECT state,message FROM steps").Scan(&state, &message); err != nil || state != "failed" || message != "notification service unavailable" {
		t.Fatal(state, message, err)
	}
	e.actionRunner = func(context.Context, Action, Event) error { t.Error("retried failed notification"); return nil }
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
}
