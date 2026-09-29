package core

import (
	"context"
	"encoding/json"
	"testing"

	"quatrro.local/automations/internal/broker"
	"quatrro.local/automations/internal/local"
)

func adminConfig() Config {
	c := EmptyConfig()
	c.Actions = []Action{{ID: "admin", Kind: "system-service", Unit: "backup.service", Operation: "restart"}}
	c.Flows = []Flow{{ID: "flow", Enabled: true, Source: "local:test", Steps: []string{"admin"}}}
	return c
}

func TestAdministrativeActionValidationAndSimulation(t *testing.T) {
	c := adminConfig()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	original := canonicalHash(capabilities(c))
	c.Actions[0].Operation = "stop"
	if original == canonicalHash(capabilities(c)) {
		t.Fatal("operation not bound to permission")
	}
	for _, a := range []Action{{Kind: "system-service", Unit: "*.service", Operation: "restart"}, {Kind: "system-service", Unit: "backup.service", Operation: "enable"}, {Kind: "system-service", Unit: "{{data.unit}}", Operation: "restart"}, {Kind: "system-service", Unit: "backup.service", Operation: "restart", Executable: "/bin/sh"}, {Kind: "system-service", Unit: "backup.service", Operation: "restart", Args: []string{"--root=/tmp"}}} {
		a.ID = "admin"
		c.Actions[0] = a
		if c.Validate() == nil {
			t.Fatal("unsafe administrative action accepted")
		}
	}
	e := testEngine(t)
	ctx := context.Background()
	c = adminConfig()
	activateAdministrativeTest(t, e, c)
	e.brokerCaller = func(context.Context, broker.Request) (broker.Result, error) {
		t.Fatal("simulation contacted privileged broker")
		return broker.Result{}, nil
	}
	raw, _ := json.Marshal(Event{Source: "local:test", Type: "test"})
	result, err := e.simulate(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var decoded map[string]any
	json.Unmarshal(encoded, &decoded)
	if decoded["effects_executed"] != false {
		t.Fatal("simulation claims effects")
	}
	step := decoded["matches"].([]any)[0].(map[string]any)["steps"].([]any)[0].(map[string]any)
	if step["operation"] != "restart" || step["resource"] != "backup.service" || step["administrative_broker_required"] != true {
		t.Fatal("simulation missing administrative scope")
	}
}

func TestAdministrativeWorkerOutcomesAndRevocation(t *testing.T) {
	for _, tc := range []struct {
		name, state string
		failure     error
		revoke      bool
	}{{"success", "completed", nil, false}, {"denied", "denied", broker.ErrDenied, false}, {"uncertain", "uncertain", broker.ErrOutcomeUnknown, false}, {"unavailable", "failed", broker.ErrUnavailable, false}, {"revoked", "denied", nil, true}} {
		t.Run(tc.name, func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			activateAdministrativeTest(t, e, adminConfig())
			calls := 0
			e.brokerCaller = func(_ context.Context, r broker.Request) (broker.Result, error) {
				calls++
				if r.Unit != "backup.service" || r.Operation != "restart" || r.Version != broker.Version || r.ID == "" {
					t.Fatal("wrong broker request")
				}
				return broker.Result{ID: r.ID, Unit: r.Unit, State: "completed"}, tc.failure
			}
			if _, err := e.ingest(ctx, Event{Source: "local:test", Type: "test"}); err != nil {
				t.Fatal(err)
			}
			if tc.revoke {
				if _, err := e.revoke(ctx, []byte(`{"scope":"flow:flow:action:admin"}`)); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < 4; i++ {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var state string
			if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil {
				t.Fatal(err)
			}
			expected := 1
			if tc.revoke {
				expected = 0
			}
			if state != tc.state || calls != expected {
				t.Fatalf("state=%s calls=%d", state, calls)
			}
		})
	}
}

func TestAdministrativeImportDoesNotGrantOrEnable(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	raw, _ := json.Marshal(adminConfig())
	if _, err := e.importConfig(ctx, raw); err != nil {
		t.Fatal(err)
	}
	draft, err := e.config(ctx, "draft")
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Actions) != 1 || draft.Actions[0].Kind != "system-service" || draft.Flows[0].Enabled {
		t.Fatal("import lost resource or enabled administrative flow")
	}
	var grants int
	if err = e.db.QueryRow("SELECT count(*) FROM grants").Scan(&grants); err != nil || grants != 0 {
		t.Fatal("import granted permissions")
	}
	e.brokerCaller = func(context.Context, broker.Request) (broker.Result, error) {
		t.Fatal("import invoked broker")
		return broker.Result{}, nil
	}
	event, _ := json.Marshal(Event{Source: "local:test", Type: "test"})
	if _, err = e.simulate(ctx, event); err != nil {
		t.Fatal(err)
	}
	var executions int
	if err = e.db.QueryRow("SELECT count(*) FROM executions").Scan(&executions); err != nil || executions != 0 {
		t.Fatal("import/simulation created effects")
	}
}

func TestAdministrativeCheckDoesNotGrantOrExecute(t *testing.T) {
	for _, scenario := range []string{"authorized", "denied", "unavailable", "unsupported", "unexpected-result"} {
		t.Run(scenario, func(t *testing.T) {
			e := testEngine(t)
			calls := 0
			e.brokerCaller = func(_ context.Context, r broker.Request) (broker.Result, error) {
				calls++
				if !r.CheckOnly || r.Operation != "restart" || r.Unit != "backup.service" {
					t.Fatal("unsafe check request")
				}
				result := broker.Result{ID: r.ID, Unit: r.Unit, State: "authorized"}
				switch scenario {
				case "denied":
					return result, broker.ErrDenied
				case "unavailable":
					return result, broker.ErrUnavailable
				case "unsupported":
					return result, broker.ErrRequest
				case "unexpected-result":
					result.State = "completed"
				}
				return result, nil
			}
			result, err := e.Handle(context.Background(), local.Request{Op: "administration.check", Data: []byte(`{"unit":"backup.service","operation":"restart"}`)})
			if err != nil {
				t.Fatal(err)
			}
			state := scenario
			if scenario == "unexpected-result" {
				state = "unavailable"
			}
			if result.(map[string]any)["state"] != state || result.(map[string]any)["effects_executed"] != false || calls != 1 {
				t.Fatal(result)
			}
			var count int
			for _, table := range []string{"grants", "executions"} {
				if err = e.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
					t.Fatalf("state changed: %s %v", table, err)
				}
			}
			if _, err = e.checkAdministrative(context.Background(), []byte(`{"unit":"*.service","operation":"restart"}`)); err == nil || calls != 1 {
				t.Fatal("invalid check reached broker")
			}
		})
	}
}

func activateAdministrativeTest(t *testing.T, e *Engine, c Config) {
	t.Helper()
	e.brokerCaller = func(_ context.Context, r broker.Request) (broker.Result, error) {
		if !r.CheckOnly {
			t.Fatal("activation executed administrative action")
		}
		return broker.Result{ID: r.ID, Unit: r.Unit, State: "authorized"}, nil
	}
	activateTest(t, e, c)
	e.brokerCaller = nil
}

func TestAdministrativeActivationRequiresFreshAuthorization(t *testing.T) {
	for _, failure := range []error{nil, broker.ErrDenied, broker.ErrUnavailable, broker.ErrRequest} {
		e := testEngine(t)
		ctx := context.Background()
		c := adminConfig()
		calls := 0
		e.brokerCaller = func(_ context.Context, r broker.Request) (broker.Result, error) {
			calls++
			if !r.CheckOnly {
				t.Fatal("effect during activation")
			}
			return broker.Result{ID: r.ID, Unit: r.Unit, State: "authorized"}, failure
		}
		raw, _ := json.Marshal(c)
		if _, err := e.saveDraft(ctx, raw); err != nil {
			t.Fatal(err)
		}
		raw, _ = json.Marshal(map[string]any{"hash": canonicalHash(c), "grants": capabilities(c)})
		_, err := e.activate(ctx, raw)
		if (err != nil) != (failure != nil) || calls != 1 {
			t.Fatalf("activation err=%v calls=%d", err, calls)
		}
		if failure != nil {
			var count int
			if err = e.db.QueryRow("SELECT count(*) FROM grants").Scan(&count); err != nil || count != 0 {
				t.Fatal("failed preflight stored grants")
			}
		}
		c.Flows[0].Enabled = false
		calls = 0
		if err = e.preflight(ctx, c); err != nil || calls != 0 {
			t.Fatal("disabled flow requires broker")
		}
	}
}
