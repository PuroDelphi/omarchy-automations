package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func instant(t *testing.T, value string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCalendarDSTAndWeekdays(t *testing.T) {
	cases := []struct {
		at, zone, after, last, want string
		days                        []string
	}{
		{"09:00", "America/Bogota", "2026-09-28T13:59:00Z", "", "2026-09-28T14:00:00Z", nil},
		{"09:00", "UTC", "2026-09-28T10:00:00Z", "", "2026-10-05T09:00:00Z", []string{"mon"}},
		{"02:30", "America/New_York", "2026-03-08T05:00:00Z", "", "2026-03-09T06:30:00Z", nil},
		{"01:30", "America/New_York", "2026-11-01T04:00:00Z", "", "2026-11-01T05:30:00Z", nil},
		{"01:30", "America/New_York", "2026-11-01T05:30:00Z", "2026-11-01T01:30", "2026-11-02T06:30:00Z", nil},
		{"01:30", "America/New_York", "2026-11-01T05:40:00Z", "", "2026-11-02T06:30:00Z", nil},
	}
	for _, tc := range cases {
		timer := Timer{Kind: "calendar", At: tc.at, Timezone: tc.zone, Weekdays: tc.days, Missed: "coalesce"}
		if err := timer.Validate(); err != nil {
			t.Fatal(err)
		}
		got, err := nextTimer(timer, instant(t, tc.after), tc.last)
		if err != nil || !got.Equal(instant(t, tc.want)) {
			t.Fatalf("%+v: got %v, %v", tc, got, err)
		}
	}
}

func timerConfig(policy string) Config {
	c := notificationConfig()
	c.Timers = []Timer{{ID: "backup", Kind: "interval", Interval: 10, Missed: policy, Enabled: true}}
	c.Flows[0].Source = "timer:backup"
	return c
}

func TestTimerCatchupAtomicityRestartAndRevocation(t *testing.T) {
	ctx := context.Background()
	e := testEngine(t)
	c := timerConfig("coalesce")
	activateTest(t, e, c)
	start := time.Unix(1800000000, 0)
	tick := func(now time.Time) {
		t.Helper()
		if err := e.scheduleTimers(ctx, now); err != nil {
			t.Fatal(err)
		}
	}
	tick(start)
	var next int64
	if err := e.db.QueryRow("SELECT next_at FROM timer_state").Scan(&next); err != nil || next != start.Unix()+10 {
		t.Fatal(next, err)
	}
	// A failure writing scheduler state must also roll back the queued event.
	if _, err := e.db.Exec(`CREATE TRIGGER fail_timer BEFORE UPDATE ON timer_state BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if err := e.scheduleTimers(ctx, start.Add(10*time.Second)); err == nil {
		t.Fatal("expected transaction failure")
	}
	var count int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 0 {
		t.Fatal("event escaped rollback")
	}
	e.db.Exec("DROP TRIGGER fail_timer")
	tick(start.Add(100 * time.Second))
	e.db.QueryRow("SELECT count(*) FROM executions").Scan(&count)
	if count != 1 {
		t.Fatal("catchup storm", count)
	}
	e.db.QueryRow("SELECT next_at FROM timer_state").Scan(&next)
	if next != start.Unix()+110 {
		t.Fatal("interval phase drift", next)
	}
	// Open another engine after closing the original to prove persisted scheduling.
	p := e.paths
	e.Close()
	reopened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	e = reopened
	t.Cleanup(func() { reopened.Close() })
	tick(start.Add(100 * time.Second))
	tick(start.Add(-100 * time.Second))
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 1 {
		t.Fatal("restart or backward clock repeated event")
	}
	if _, err = e.revoke(ctx, []byte(`{"scope":"timer:backup"}`)); err != nil {
		t.Fatal(err)
	}
	tick(start.Add(110 * time.Second))
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 1 {
		t.Fatal("revoked timer emitted")
	}
	activateTest(t, e, c)
	tick(start.Add(120 * time.Second))
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 2 {
		t.Fatal("reapproved timer failed", count)
	}
}

func TestTimerSkipAdmissionAndImport(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := timerConfig("skip")
	activateTest(t, e, c)
	start := time.Unix(1800000000, 0)
	if err := e.scheduleTimers(ctx, start); err != nil {
		t.Fatal(err)
	}
	if err := e.scheduleTimers(ctx, start.Add(29*time.Second)); err != nil {
		t.Fatal(err)
	}
	var count int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 0 {
		t.Fatal("skip emitted late event")
	}
	if _, err := e.control(ctx, []byte(`{"paused":true,"admission":"reject"}`)); err != nil {
		t.Fatal(err)
	}
	if err := e.scheduleTimers(ctx, start.Add(30*time.Second)); err == nil {
		t.Fatal("admission pause ignored")
	}
	var next int64
	e.db.QueryRow("SELECT next_at FROM timer_state").Scan(&next)
	if next != start.Unix()+30 {
		t.Fatal("failed admission consumed occurrence")
	}
	e.control(ctx, []byte(`{"paused":true,"admission":"retain"}`))
	if err := e.scheduleTimers(ctx, start.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	e.db.QueryRow("SELECT count(*) FROM executions WHERE state='pending'").Scan(&count)
	if count != 1 {
		t.Fatal("paused effects did not retain event")
	}
	raw, _ := json.Marshal(c)
	if _, err := e.importConfig(ctx, raw); err != nil {
		t.Fatal(err)
	}
	draft, err := e.config(ctx, "draft")
	if err != nil || draft.Timers[0].Enabled {
		t.Fatal("import enabled timer", err)
	}
}

func TestTimerValidationAndMigration(t *testing.T) {
	for _, timer := range []Timer{
		{Kind: "interval", Interval: 0, Missed: "skip"},
		{Kind: "interval", Interval: 10, Missed: "all"},
		{Kind: "interval", Interval: 10, Missed: "skip", At: "09:00"},
		{Kind: "calendar", At: "25:00", Timezone: "UTC", Missed: "skip"},
		{Kind: "calendar", At: "09:00", Timezone: "Local", Missed: "skip"},
		{Kind: "calendar", At: "09:00", Timezone: "Missing/Zone", Missed: "skip"},
		{Kind: "calendar", At: "09:00", Timezone: "UTC", Weekdays: []string{"mon", "mon"}, Missed: "skip"},
	} {
		if err := timer.Validate(); err == nil {
			t.Fatal("invalid timer accepted", timer)
		}
	}
	c := notificationConfig()
	c.Flows[0].Source = "timer:missing"
	if err := c.Validate(); err == nil {
		t.Fatal("unregistered timer source accepted")
	}
	e := testEngine(t)
	if _, err := e.db.Exec("ALTER TABLE executions DROP COLUMN context;DROP TABLE timer_state;DELETE FROM schema_version;INSERT INTO schema_version VALUES(2)"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(context.Background(), e.db); err != nil {
		t.Fatal(err)
	}
	var version int
	e.db.QueryRow("SELECT version FROM schema_version").Scan(&version)
	if version != DatabaseVersion {
		t.Fatal(version)
	}
	if _, err := e.db.Exec("SELECT * FROM timer_state"); err != nil {
		t.Fatal(err)
	}
}

func TestCalendarSchedulerEmitsOnceDuringRepeatedHour(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := timerConfig("coalesce")
	c.Timers[0] = Timer{ID: "backup", Kind: "calendar", At: "01:30", Timezone: "America/New_York", Missed: "coalesce", Enabled: true}
	activateTest(t, e, c)
	for _, stamp := range []string{"2026-11-01T04:00:00Z", "2026-11-01T05:30:00Z", "2026-11-01T06:30:00Z"} {
		if err := e.scheduleTimers(ctx, instant(t, stamp)); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	e.db.QueryRow("SELECT count(*) FROM executions").Scan(&count)
	if count != 1 {
		t.Fatal("repeated local time produced duplicate execution", count)
	}
	// Editing the schedule resets its next occurrence and invalidates its grant.
	c.Timers[0].At = "02:00"
	activateTest(t, e, c)
	if err := e.scheduleTimers(ctx, instant(t, "2026-11-01T06:31:00Z")); err != nil {
		t.Fatal(err)
	}
	var next int64
	e.db.QueryRow("SELECT next_at FROM timer_state").Scan(&next)
	if next != instant(t, "2026-11-01T07:00:00Z").Unix() {
		t.Fatal("changed calendar not applied", next)
	}
}
