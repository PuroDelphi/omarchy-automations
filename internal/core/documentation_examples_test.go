package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Exercise the files shipped to users, including resolved templates, without
// granting permissions or producing desktop/network effects.
func TestDocumentedUseCases(t *testing.T) {
	cases := []struct {
		file     string
		event    Event
		wantBody string
	}{
		{"webhook-notification", Event{Source: "entry:deploy", Type: "webhook", Data: map[string]any{"state": "failed", "message": "Build failed"}}, "Build failed"},
		{"monitor-outbound", Event{Source: "monitor:disk-space", Type: "alert", Data: map[string]any{"metric": "disk", "value": float64(8), "state": "alert"}}, `{"metric":"disk","state":"alert","value":8}`},
		{"monitor-outbound", Event{Source: "monitor:disk-space", Type: "recovered", Data: map[string]any{"metric": "disk", "value": float64(18), "state": "recovered"}}, `{"metric":"disk","state":"recovered","value":18}`},
		{"scheduled-notification", Event{Source: "timer:weekday-reminder", Type: "scheduled", Data: map[string]any{"scheduled_at": "2026-09-28T14:00:00Z"}}, "Scheduled reminder: 2026-09-28T14:00:00Z"},
	}
	for _, tc := range cases {
		t.Run(tc.file+"/"+tc.event.Type, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "use-cases", tc.file+".json"))
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
			for _, entry := range c.Entries {
				if entry.Enabled {
					t.Fatal("example entry enabled")
				}
			}
			for _, monitor := range c.Monitors {
				if monitor.Enabled {
					t.Fatal("example monitor enabled")
				}
			}
			for _, timer := range c.Timers {
				if timer.Enabled {
					t.Fatal("example timer enabled")
				}
			}
			for i, flow := range c.Flows {
				if flow.Enabled {
					t.Fatal("example flow enabled")
				}
				c.Flows[i].Enabled = true
			}
			e := testEngine(t)
			ctx := context.Background()
			e.actionRunner = func(context.Context, Action, Event) error {
				t.Fatal("documentation simulation executed an action")
				return nil
			}
			e.deliverySender = func(context.Context, Destination, string, []byte) deliveryResult {
				t.Fatal("documentation simulation sent HTTP")
				return deliveryResult{}
			}
			raw, _ = json.Marshal(c)
			if _, err = e.saveDraft(ctx, raw); err != nil {
				t.Fatal(err)
			}
			raw, _ = json.Marshal(tc.event)
			result, err := e.simulate(ctx, raw)
			if err != nil {
				t.Fatal(err)
			}
			report := result.(map[string]any)
			matches := report["matches"].([]map[string]any)
			if report["effects_executed"] != false || len(matches) != 1 {
				t.Fatal(report)
			}
			steps := matches[0]["steps"].([]map[string]any)
			if len(steps) != 1 || steps[0]["body"] != tc.wantBody {
				t.Fatal(steps)
			}
			for _, table := range []string{"executions", "outbox", "grants"} {
				var n int
				if err = e.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
					t.Fatal(table, n, err)
				}
			}
		})
	}
}

func documentedCodeConfig(t *testing.T, name string) Config {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "use-cases", name+".json"))
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
	if len(c.Scripts) > 0 {
		source, err := os.ReadFile("../../examples/scripts/check-project.py")
		if err != nil || string(source) != c.Scripts[0].Code {
			t.Fatal("script source differs from import", err)
		}
	}
	if len(c.Adapters) > 0 {
		source, err := os.ReadFile("../../examples/adapters/status-normalizer.py")
		if err != nil || string(source) != c.Adapters[0].Code {
			t.Fatal("adapter source differs from import", err)
		}
	}
	return c
}

func TestDocumentedCodeExamples(t *testing.T) {
	for _, name := range []string{"typed-script", "adapter-notification"} {
		t.Run(name, func(t *testing.T) {
			c := documentedCodeConfig(t, name)
			e := testEngine(t)
			ctx := context.Background()
			raw, _ := json.Marshal(c)
			if _, err := e.saveDraft(ctx, raw); err != nil {
				t.Fatal(err)
			}
			data := map[string]any{"project": "demo-project", "count": float64(3)}
			if name == "adapter-notification" {
				data = map[string]any{"status": "failed", "message": "Build failed", "adapter": map[string]any{"severity": "forged"}}
			}
			raw, _ = json.Marshal(Event{Source: c.Flows[0].Source, Type: "test", Data: data})
			value, err := e.simulate(ctx, raw)
			if err != nil {
				t.Fatal(err)
			}
			report := value.(map[string]any)
			steps := report["matches"].([]map[string]any)[0]["steps"].([]map[string]any)
			if report["effects_executed"] != false {
				t.Fatal(report)
			}
			if name == "typed-script" {
				args := steps[0]["arguments"].([]string)
				if len(args) != 3 || args[0] != "demo-project" || args[1] != "3" || args[2] != "true" {
					t.Fatal(args)
				}
				data["count"] = float64(11)
				raw, _ = json.Marshal(Event{Source: c.Flows[0].Source, Data: data})
				if _, err = e.simulate(ctx, raw); err == nil {
					t.Fatal("out-of-contract example accepted")
				}
			} else if steps[0]["result_available"] != false || steps[1]["requires_adapter_result"] != true {
				t.Fatal(steps)
			}
			for _, table := range []string{"executions", "outbox", "grants"} {
				var n int
				if err = e.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
					t.Fatal(table, n, err)
				}
			}
		})
	}
}

func TestHostDocumentedCodeExamples(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("host test")
	}
	ctx := context.Background()
	c := documentedCodeConfig(t, "typed-script")
	e := testEngine(t)
	for _, data := range []map[string]any{{"project": "demo-project", "count": float64(3)}, {"project": "release-2", "count": float64(10)}} {
		if err := e.performScript(ctx, c, c.Actions[0], Event{Data: data}); err != nil {
			t.Fatal(err)
		}
	}
	c = documentedCodeConfig(t, "adapter-notification")
	for _, tc := range []struct{ status, message, want string }{{"failed", "Build failed", "error: Build failed"}, {"passed", "Build passed", "info: Build passed"}} {
		ev := Event{Source: "local:build", Type: "build", Data: map[string]any{"status": tc.status, "message": tc.message}}
		data, err := executeAdapter(ctx, c.Adapters[0], ev)
		if err != nil {
			t.Fatal(err)
		}
		ev.Data["adapter"] = data
		body, err := render(c.Actions[1].Body, ev)
		if err != nil || body != tc.want {
			t.Fatal(body, err)
		}
	}
}

// Keep all documented metric variants importable and verify the boundary
// behavior described in the recipes without sampling the user's machine.
func TestDocumentedMonitorVariants(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "use-cases", "monitor-variants.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err := decode(raw, &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(c.Flows) != 0 || len(c.Actions) != 0 {
		t.Fatal("monitor recipes must not perform effects")
	}
	counts := map[string]int{}
	for _, m := range c.Monitors {
		if m.Enabled {
			t.Fatalf("%s starts enabled", m.ID)
		}
		counts[m.Metric]++
		if m.Metric == "journal" {
			continue
		}
		// At the exact threshold, no pending alert starts.
		state, event := transition(m, monitorState{}, m.Threshold, 1000)
		if event != "" || state.Phase != "normal" {
			t.Fatalf("%s: alert boundary", m.ID)
		}
		// Equality at recovery must preserve an existing alert.
		state, event = transition(m, monitorState{Phase: "alert"}, m.Recovery, 1000)
		if event != "" || state.Phase != "alert" {
			t.Fatalf("%s: recovery boundary", m.ID)
		}
	}
	for _, metric := range []string{"cpu", "memory", "disk", "battery", "service", "process", "file_exists", "file_age", "file_size", "connectivity", "temperature", "journal"} {
		if counts[metric] != 2 {
			t.Fatalf("%s needs two recipes", metric)
		}
	}
	if len(c.Monitors) != 24 {
		t.Fatal("unexpected recipe count")
	}
}
