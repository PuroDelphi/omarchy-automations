package core

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHistoryNormalizesLegacyFinishedStepsOnly(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	// Legacy successful-step label and a genuinely pending delivery are separate.
	for step, state := range []string{"pending", "failed", "denied", "uncertain", "completed"} {
		if _, err := e.db.Exec("INSERT INTO steps(execution,step,state,message,finished_at) VALUES('legacy',?,?, 'redacted', '2026-09-28T00:00:00Z')", step, state); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.db.Exec("INSERT INTO outbox(execution,step,body,attempts,next_at,created_at,state) VALUES('legacy',5,'{}',1,99,1,'pending')"); err != nil {
		t.Fatal(err)
	}
	result, err := e.detail(ctx, []byte(`{"id":"legacy"}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	var report struct {
		Steps []struct {
			State string `json:"state"`
		} `json:"steps"`
		Deliveries []struct {
			State string `json:"state"`
		} `json:"deliveries"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"completed", "failed", "denied", "uncertain", "completed"} {
		if report.Steps[i].State != want {
			t.Fatal(report)
		}
	}
	if len(report.Deliveries) != 1 || report.Deliveries[0].State != "pending" {
		t.Fatal(report)
	}
	var stored string
	if err := e.db.QueryRow("SELECT state FROM steps WHERE execution='legacy' AND step=0").Scan(&stored); err != nil || stored != "pending" {
		t.Fatal("read rewrote history", stored, err)
	}
}
