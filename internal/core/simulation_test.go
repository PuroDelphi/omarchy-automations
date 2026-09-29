package core

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSimulationExplainsRejectedFlowsWithoutEffectsOrSecrets(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	c.Destinations = []Destination{{ID: "out", URL: "https://example.test", Method: "POST", Auth: "bearer", Secret: "unavailable-credential"}}
	c.Actions = []Action{{ID: "notice", Kind: "http", Destination: "out", Body: `{"message":"{{data.message}}"}`}}
	c.Flows[0].Conditions = []Condition{{Field: "data.status", Op: "eq", Value: "failed"}}
	for _, id := range []string{"disabled", "source", "condition", "missing"} {
		f := c.Flows[0]
		f.ID = id
		switch id {
		case "disabled":
			f.Enabled = false
		case "source":
			f.Source = "local:other"
		case "condition":
			f.Conditions = []Condition{{Field: "data.status", Op: "eq", Value: "ok"}}
		case "missing":
			f.Conditions = []Condition{{Field: "data.absent", Op: "ne", Value: "ok"}}
		}
		c.Flows = append(c.Flows, f)
	}
	raw, _ := json.Marshal(c)
	if _, err := e.saveDraft(ctx, raw); err != nil {
		t.Fatal(err)
	}
	e.actionRunner = func(context.Context, Action, Event) error { t.Fatal("simulation executed action"); return nil }
	e.deliverySender = func(context.Context, Destination, string, []byte) deliveryResult {
		t.Fatal("simulation sent HTTP")
		return deliveryResult{}
	}
	raw, _ = json.Marshal(Event{Source: "local:test", Data: map[string]any{"message": "preview", "status": "failed"}})
	value, err := e.simulate(ctx, raw)
	if err != nil {
		t.Fatal("simulation tried to resolve unavailable credential", err)
	}
	report := value.(map[string]any)
	evaluations := report["evaluations"].([]map[string]any)
	if report["effects_executed"] != false || len(evaluations) != 5 {
		t.Fatal(report)
	}
	for i, evaluation := range evaluations {
		if evaluation["matched"] != (i == 0) {
			t.Fatal(evaluation)
		}
	}
	if evaluations[1]["enabled"] != false || evaluations[2]["source_matched"] != false {
		t.Fatal(evaluations)
	}
	for _, i := range []int{3, 4} {
		if evaluations[i]["conditions"].([]map[string]any)[0]["matched"] != false {
			t.Fatal(evaluations[i])
		}
	}
	matches := report["matches"].([]map[string]any)
	if len(matches) != 1 || matches[0]["steps"].([]map[string]any)[0]["body"] != `{"message":"preview"}` {
		t.Fatal(matches)
	}
	for _, table := range []string{"executions", "outbox", "grants"} {
		var count int
		if err := e.db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatal(table, count, err)
		}
	}
}
