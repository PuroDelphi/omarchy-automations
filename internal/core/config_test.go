package core

import (
	"context"
	"encoding/json"
	"path/filepath"
	"quatrro.local/automations/internal/paths"
	"testing"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	base := t.TempDir()
	p := paths.Paths{Config: filepath.Join(base, "config"), State: filepath.Join(base, "state"), Runtime: filepath.Join(base, "runtime")}
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}
	e, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}
func notificationConfig() Config {
	c := EmptyConfig()
	c.Actions = []Action{{ID: "notice", Kind: "notify", Title: "Deployment", Body: "Failed"}}
	c.Flows = []Flow{{ID: "deploy", Source: "local:test", Enabled: true, Steps: []string{"notice"}}}
	return c
}
func TestRevisionGrantAndRevocation(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	b, _ := json.Marshal(c)
	if _, err := e.saveDraft(ctx, b); err != nil {
		t.Fatal(err)
	}
	req := map[string]any{"hash": canonicalHash(c), "grants": capabilities(c)}
	b, _ = json.Marshal(req)
	if _, err := e.activate(ctx, b); err != nil {
		t.Fatal(err)
	}
	original := canonicalHash(c)
	c.Actions[0].Body = "Changed"
	newBytes, _ := json.Marshal(c)
	if _, err := e.saveDraft(ctx, newBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := e.activate(ctx, b); err == nil {
		t.Fatal("accepted stale draft")
	}
	req["hash"] = canonicalHash(c)
	b, _ = json.Marshal(req)
	if _, err := e.activate(ctx, b); err == nil {
		t.Fatal("reused grant after action changed")
	}
	active, err := e.config(ctx, "active")
	if err != nil || canonicalHash(active) != original {
		t.Fatal("active revision changed without authorization", err)
	}
	if _, err = e.revoke(ctx, []byte(`{"scope":"flow:deploy:action:notice"}`)); err != nil {
		t.Fatal(err)
	}
	var n int
	e.db.QueryRow("SELECT count(*) FROM grants").Scan(&n)
	if n != 0 {
		t.Fatal("revocation failed")
	}
	var stored string
	if err = e.db.QueryRow("SELECT config FROM revisions WHERE hash=?", original).Scan(&stored); err != nil {
		t.Fatal("immutable revision lost", err)
	}
}
func TestRejectAmbiguousOrUnsafeConfig(t *testing.T) {
	cases := []func(*Config){func(c *Config) { c.Version = 7 }, func(c *Config) { c.Flows[0].Steps = []string{"missing"} }, func(c *Config) {
		c.Actions[0] = Action{ID: "notice", Kind: "command", Executable: "/bin/echo", Args: []string{"{{data.command}}"}, Timeout: 5}
	}, func(c *Config) { c.Destinations = []Destination{{ID: "out", URL: "http://127.0.0.1", Method: "POST"}} }}
	for i, mutate := range cases {
		c := notificationConfig()
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("accepted invalid case %d", i)
		}
	}
	c := notificationConfig()
	raw, _ := json.Marshal(c)
	var out Config
	if err := decode(append(raw, []byte(` {}`)...), &out); err == nil {
		t.Fatal("accepted trailing JSON")
	}
}

func TestCapabilityNamespacesCannotCollide(t *testing.T) {
	c := notificationConfig()
	c.Flows[0].ID = "monitor"
	c.Monitors = []Monitor{{ID: "notice", Metric: "memory", Enabled: true}}
	grants := capabilities(c)
	if len(grants) != 2 || grants["monitor:notice"] == "" || grants["flow:monitor:action:notice"] == "" {
		t.Fatalf("action overwrote independent monitor permission: %v", grants)
	}
}
