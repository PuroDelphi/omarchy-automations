package core

import (
	"context"
	"testing"
	"time"
)

func TestMonitorDurationHysteresisAndRecovery(t *testing.T) {
	m := Monitor{Metric: "disk", Threshold: 10, Recovery: 15, Duration: 30, Interval: 5, Cooldown: 60}
	s := monitorState{}
	var event string
	s, event = transition(m, s, 9, 100)
	if event != "" || s.Phase != "pending" {
		t.Fatal(s, event)
	}
	s, event = transition(m, s, 8, 129)
	if event != "" {
		t.Fatal("alert too early")
	}
	s, event = transition(m, s, 8, 130)
	if event != "alert" {
		t.Fatal("no alert")
	}
	s, event = transition(m, s, 8, 140)
	if event != "" {
		t.Fatal("repeated alert")
	}
	s, event = transition(m, s, 12, 150)
	if event != "" || s.Phase != "alert" {
		t.Fatal("hysteresis ignored")
	}
	s, event = transition(m, s, 20, 155)
	if event != "recovered" {
		t.Fatal("no recovery")
	}
	s, event = transition(m, s, 9, 160)
	if event != "" {
		t.Fatal("cooldown ignored")
	}
	s, event = transition(m, s, 9, 190)
	if event != "alert" {
		t.Fatal("second episode missing")
	}
}

func TestMonitorEventAndStateAreAtomic(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	m := Monitor{ID: "disk", Metric: "disk", Path: "/", Threshold: 10, Recovery: 15, Duration: 0, Interval: 5, Cooldown: 10, Enabled: true}
	c.Monitors = []Monitor{m}
	c.Flows[0].Source = "monitor:disk"
	activateTest(t, e, c)
	s := monitorState{Hash: canonicalHash(m), Phase: "alert", Value: 1, Last: 100, Next: 105}
	if _, err := e.db.Exec(`CREATE TRIGGER prevent_monitor BEFORE INSERT ON monitor_state BEGIN SELECT RAISE(ABORT,'injected failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := e.commitMonitor(ctx, m, s, "alert", time.Unix(100, 0)); err == nil {
		t.Fatal("expected injected failure")
	}
	var n int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&n)
	if n != 0 {
		t.Fatal("event persisted without monitor transition")
	}
	e.db.Exec("DROP TRIGGER prevent_monitor")
	if err := e.commitMonitor(ctx, m, s, "alert", time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	e.db.QueryRow("SELECT count(*) FROM executions").Scan(&n)
	if n != 1 {
		t.Fatal("missing monitor execution")
	}
}

func TestRecoveryNeedsContinuousDuration(t *testing.T) {
	m := Monitor{Metric: "cpu", Threshold: 90, Recovery: 70, RecoveryDuration: 20, Interval: 5, Cooldown: 5}
	s := monitorState{Phase: "alert"}
	var event string
	s, event = transition(m, s, 60, 100)
	if event != "" {
		t.Fatal("recovered early")
	}
	s, event = transition(m, s, 80, 110)
	if event != "" || s.Since != 0 {
		t.Fatal("recovery timer not reset")
	}
	s, event = transition(m, s, 60, 120)
	if event != "" {
		t.Fatal("recovered early after reset")
	}
	s, event = transition(m, s, 60, 140)
	if event != "recovered" {
		t.Fatal("no recovery")
	}
}
