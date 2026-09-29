package core

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

const adapterTransform = "import json,sys\nr=json.load(sys.stdin)\nprint(json.dumps({'version':1,'id':r['id'],'data':{'message':r['event']['data']['message'].upper()}}))\n"

func adapterConfig(code string) Config {
	c := notificationConfig()
	adapter := Adapter{Manifest: adapterManifest(), Code: code}
	adapter.Revision = adapterRevision(adapter)
	c.Adapters = []Adapter{adapter}
	c.Actions = append([]Action{{ID: "normalize", Kind: "adapter", Adapter: adapter.ID, AdapterRevision: adapter.Revision}}, c.Actions...)
	c.Flows[0].Steps = []string{"normalize", "notice"}
	c.Actions[1].Body = "{{data.adapter.message}}"
	return c
}

func TestAdapterReferenceAndSimulation(t *testing.T) {
	c := adapterConfig(adapterTransform)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	old := capabilities(c)[actionScope(c.Flows[0].ID, "normalize")]
	c.Adapters[0].OutputLimit = 2048
	c.Adapters[0].Revision = adapterRevision(c.Adapters[0])
	if c.Validate() == nil {
		t.Fatal("stale adapter action accepted")
	}
	c.Actions[0].AdapterRevision = c.Adapters[0].Revision
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if capabilities(c)[actionScope(c.Flows[0].ID, "normalize")] == old {
		t.Fatal("manifest change retained permission")
	}
	e := testEngine(t)
	raw, _ := json.Marshal(c)
	if _, err := e.saveDraft(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(Event{Source: c.Flows[0].Source, Data: map[string]any{"message": "hello", "adapter": map[string]any{"message": "spoofed"}}})
	result, err := e.simulate(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	steps := result.(map[string]any)["matches"].([]map[string]any)[0]["steps"].([]map[string]any)
	if steps[0]["result_available"] != false || steps[1]["requires_adapter_result"] != true {
		t.Fatal(steps)
	}
}

func TestHostAdapterResultFeedsNextStepAfterRestart(t *testing.T) {
	hostIsolation(t)
	e := testEngine(t)
	ctx := context.Background()
	c := adapterConfig(adapterTransform)
	activateTest(t, e, c)
	if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source, Data: map[string]any{"message": "hello"}}); err != nil {
		t.Fatal(err)
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	var step int
	var state, raw string
	if err := e.db.QueryRow("SELECT step,state,context FROM executions").Scan(&step, &state, &raw); err != nil || step != 1 || state != "pending" || !strings.Contains(raw, "HELLO") {
		t.Fatal(step, state, raw, err)
	}
	paths := e.paths
	e.Close()
	next, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	calls := 0
	next.actionRunner = func(_ context.Context, a Action, event Event) error {
		text, err := render(a.Body, event)
		if err != nil || text != "HELLO" {
			t.Errorf("next step input: %q %v", text, err)
		}
		calls++
		return nil
	}
	for range 2 {
		if err := next.tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := next.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil || state != "completed" || calls != 1 {
		t.Fatal(state, calls, err)
	}
}

func TestHostAdapterRefusalDoesNotAdvance(t *testing.T) {
	hostIsolation(t)
	for _, scenario := range []string{"revoked", "replaced-revision", "invalid-response", "output-quota", "context-quota", "input-quota", "timeout"} {
		t.Run(scenario, func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			code := adapterTransform
			if scenario == "invalid-response" {
				code = "import json,sys\nr=json.load(sys.stdin)\nprint(json.dumps({'version':2,'id':r['id'],'data':{}}))\n"
			}
			if scenario == "output-quota" {
				code = "print('x'*70000)\n"
			}
			if scenario == "timeout" {
				code = "import time\ntime.sleep(60)\n"
			}
			if scenario == "context-quota" {
				code = "import json,sys\nr=json.load(sys.stdin)\nprint(json.dumps({'version':1,'id':r['id'],'data':{'large':'x'*60000}}))\n"
			}
			c := adapterConfig(code)
			if scenario == "context-quota" {
				c.Adapters[0].OutputLimit = 65536
			}
			if scenario == "input-quota" {
				c.Adapters[0].InputLimit = 256
			}
			c.Adapters[0].Revision = adapterRevision(c.Adapters[0])
			c.Actions[0].AdapterRevision = c.Adapters[0].Revision
			activateTest(t, e, c)
			message := "hello"
			if scenario == "context-quota" {
				message = strings.Repeat("x", 220000)
			}
			if scenario == "input-quota" {
				message = strings.Repeat("x", 3000)
			}
			if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source, Data: map[string]any{"message": message}}); err != nil {
				t.Fatal(err)
			}
			expected := "failed"
			if scenario == "revoked" {
				raw, _ := json.Marshal(map[string]string{"scope": actionScope(c.Flows[0].ID, "normalize")})
				if _, err := e.revoke(ctx, raw); err != nil {
					t.Fatal(err)
				}
				expected = "denied"
			} else if scenario == "replaced-revision" {
				c.Adapters[0].OutputLimit = 2048
				c.Adapters[0].Revision = adapterRevision(c.Adapters[0])
				c.Actions[0].AdapterRevision = c.Adapters[0].Revision
				activateTest(t, e, c)
				expected = "denied"
			}
			for range 2 {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var step int
			var state, raw string
			if err := e.db.QueryRow("SELECT step,state,context FROM executions").Scan(&step, &state, &raw); err != nil || step != 0 || state != expected || raw != "" {
				t.Fatal(step, state, raw, err)
			}
		})
	}
}

func TestAdapterParentOutputBound(t *testing.T) {
	output := &adapterOutput{limit: 32}
	_, err := io.Copy(output, io.LimitReader(strings.NewReader(strings.Repeat("x", 64)), 64))
	if err == nil || output.data.Len() > 32 {
		t.Fatal("output bound bypassed", output.data.Len(), err)
	}
}
