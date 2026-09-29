package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func contextJob(t *testing.T, e *Engine) job {
	t.Helper()
	var j job
	var payload string
	if err := e.db.QueryRow(`SELECT x.id,x.step,v.payload FROM executions x JOIN events v ON v.id=x.event_id AND v.source=x.event_source WHERE x.flow='deploy'`).Scan(&j.ID, &j.Step, &payload); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(payload), &j.Event); err != nil {
		t.Fatal(err)
	}
	return j
}

func TestAdapterContextIsPrivateAndSurvivesReopen(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	c.Flows[0].Steps = []string{"notice", "notice"}
	second := c.Flows[0]
	second.ID = "other"
	second.Steps = []string{"notice"}
	c.Flows = append(c.Flows, second)
	activateTest(t, e, c)
	if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source, Data: map[string]any{"message": "original"}}); err != nil {
		t.Fatal(err)
	}
	j := contextJob(t, e)
	if err := e.finishAdapter(ctx, j, map[string]any{"message": "transformed"}); err != nil {
		t.Fatal(err)
	}
	if _, exists := j.Event.Data["adapter"]; exists {
		t.Fatal("input event mutated")
	}
	paths := e.paths
	e.Close()
	reopened, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var original, adapted int
	reopened.actionRunner = func(_ context.Context, _ Action, event Event) error {
		if event.Data["message"] != "original" {
			t.Error("original data overwritten")
		}
		if value, ok := event.Data["adapter"].(map[string]any); ok {
			if value["message"] != "transformed" {
				t.Error(value)
			}
			adapted++
		} else {
			original++
		}
		return nil
	}
	for range 4 {
		if err := reopened.tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if original != 1 || adapted != 1 {
		t.Fatal("context leaked or disappeared", original, adapted)
	}
	var raw string
	if err := reopened.db.QueryRow("SELECT payload FROM events").Scan(&raw); err != nil || strings.Contains(raw, "transformed") {
		t.Fatal(raw, err)
	}
}

func TestAdapterContextAndStepRollbackTogether(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	activateTest(t, e, c)
	if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source}); err != nil {
		t.Fatal(err)
	}
	j := contextJob(t, e)
	if _, err := e.db.Exec(`CREATE TRIGGER reject_context BEFORE UPDATE OF context ON executions BEGIN SELECT RAISE(ABORT,'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := e.finishAdapter(ctx, j, map[string]any{"message": "must not commit"}); err == nil {
		t.Fatal("injected failure ignored")
	}
	var step, count int
	var raw string
	if err := e.db.QueryRow("SELECT step,context FROM executions WHERE id=?", j.ID).Scan(&step, &raw); err != nil {
		t.Fatal(err)
	}
	if err := e.db.QueryRow("SELECT count(*) FROM steps").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if step != 0 || raw != "" || count != 0 {
		t.Fatal("partial commit", step, raw, count)
	}
}

func TestAdapterContextCountsTowardStorageQuota(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	activateTest(t, e, c)
	if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source, Data: map[string]any{"message": "original"}}); err != nil {
		t.Fatal(err)
	}
	j := contextJob(t, e)
	// A small test quota makes byte-vs-rune accounting observable without filling disk.
	var originalBytes int64
	if err := e.db.QueryRow(payloadUsageSQL).Scan(&originalBytes); err != nil {
		t.Fatal(err)
	}
	policy := defaultStoragePolicy()
	policy.MaxPayloadBytes = originalBytes + 100
	raw, _ := json.Marshal(policy)
	if _, err := e.db.Exec("INSERT INTO settings(key,value) VALUES('storage_policy',?)", string(raw)); err != nil {
		t.Fatal(err)
	}
	if err := e.finishAdapter(ctx, j, map[string]any{"message": strings.Repeat("á", 60)}); err == nil {
		t.Fatal("context bypassed quota")
	}
	var stored string
	if err := e.db.QueryRow("SELECT context FROM executions").Scan(&stored); err != nil || stored != "" {
		t.Fatal(stored, err)
	}
	policy.MaxPayloadBytes = 1 << 20
	raw, _ = json.Marshal(policy)
	if _, err := e.db.Exec("UPDATE settings SET value=? WHERE key='storage_policy'", string(raw)); err != nil {
		t.Fatal(err)
	}
	if err := e.finishAdapter(ctx, j, map[string]any{"message": "á"}); err != nil {
		t.Fatal(err)
	}
	if err := e.db.QueryRow("SELECT context FROM executions").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	status, err := e.storageStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.(map[string]any)["payload_bytes"].(int64) != originalBytes+int64(len(stored)) {
		t.Fatal(status)
	}
}

func TestVersionFourMigrationPreservesPendingJob(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	activateTest(t, e, c)
	if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec("ALTER TABLE executions DROP COLUMN context; DELETE FROM schema_version; INSERT INTO schema_version VALUES(4)"); err != nil {
		t.Fatal(err)
	}
	paths := e.paths
	e.Close()
	next, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	var version int
	var state, raw string
	if err := next.db.QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil || version != 5 {
		t.Fatal(version, err)
	}
	if err := next.db.QueryRow("SELECT state,context FROM executions").Scan(&state, &raw); err != nil || state != "pending" || raw != "" {
		t.Fatal(state, raw, err)
	}
}
