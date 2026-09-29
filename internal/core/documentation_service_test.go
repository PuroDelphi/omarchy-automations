package core

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func documentedServiceConfig(t *testing.T) Config {
	t.Helper()
	raw, err := os.ReadFile("../../examples/use-cases/user-service.json")
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
	for i, f := range c.Flows {
		if f.Enabled {
			t.Fatal("example flow enabled")
		}
		c.Flows[i].Enabled = true
	}
	return c
}

func TestDocumentedServiceSimulation(t *testing.T) {
	c := documentedServiceConfig(t)
	e := testEngine(t)
	ctx := context.Background()
	raw, _ := json.Marshal(c)
	if _, err := e.saveDraft(ctx, raw); err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"status", "restart", "stop"} {
		raw, _ = json.Marshal(Event{Source: "local:service", Data: map[string]any{"operation": op}})
		value, err := e.simulate(ctx, raw)
		if err != nil {
			t.Fatal(err)
		}
		report := value.(map[string]any)
		matches := report["matches"].([]map[string]any)
		expected := 1
		if op == "stop" {
			expected = 0
		}
		if report["effects_executed"] != false || len(matches) != expected {
			t.Fatal(report)
		}
		if expected == 1 && matches[0]["flow"] != "request-"+op {
			t.Fatal(report)
		}
	}
	var n int
	if err := e.db.QueryRow("SELECT count(*) FROM executions").Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}

func TestHostDocumentedService(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("host test")
	}
	ctx := context.Background()
	unit := "omarchy-doc-service-" + newID() + ".service"
	if err := runBounded(ctx, 5*time.Second, "systemd-run", "--user", "--unit="+unit, "--collect", "--property=Type=exec", "--", "/usr/bin/sleep", "120"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := runBounded(context.Background(), 5*time.Second, "systemctl", "--user", "stop", unit); err != nil {
			t.Error(err)
		}
	}()
	pid := func() string {
		t.Helper()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		raw, err := exec.CommandContext(ctx, "systemctl", "--user", "show", unit, "--property=MainPID", "--value").Output()
		if err != nil {
			t.Fatal(err)
		}
		value := strings.TrimSpace(string(raw))
		if value == "" || value == "0" {
			t.Fatal("service has no process")
		}
		return value
	}
	before := pid()
	c := documentedServiceConfig(t)
	for i := range c.Actions {
		c.Actions[i].Unit = unit
	}
	e := testEngine(t)
	activateTest(t, e, c)
	send := func(op string) {
		t.Helper()
		if _, err := e.ingest(ctx, Event{Source: "local:service", Type: "test", Data: map[string]any{"operation": op}}); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if err := e.tick(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
	send("status")
	if pid() != before {
		t.Fatal("status restarted process")
	}
	send("restart")
	after := pid()
	if after == before {
		t.Fatal("restart did not replace process")
	}
	if _, err := e.revoke(ctx, []byte(`{"scope":"flow:request-restart:action:service-restart"}`)); err != nil {
		t.Fatal(err)
	}
	send("restart")
	if pid() != after {
		t.Fatal("revoked action restarted process")
	}
	var completed, denied int
	if err := e.db.QueryRow("SELECT count(*) FROM executions WHERE state='completed'").Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if err := e.db.QueryRow("SELECT count(*) FROM executions WHERE state='denied'").Scan(&denied); err != nil || completed != 2 || denied != 1 {
		t.Fatal(completed, denied, err)
	}
}
