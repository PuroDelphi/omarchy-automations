package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"modernc.org/sqlite"
	"strings"
	"testing"
	"time"
)

func TestStorageQuotaRetainsDedupAndPendingJobs(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	e.actionRunner = func(context.Context, Action, Event) error { return nil }
	old := time.Now().Add(-10 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	e.ingest(ctx, Event{ID: "completed", Source: "local:test"})
	e.tick(ctx)
	e.tick(ctx)
	e.ingest(ctx, Event{ID: "pending", Source: "local:test"})
	e.db.Exec("UPDATE events SET created_at=?", old)
	e.db.Exec("UPDATE executions SET created_at=?", old)
	e.db.Exec("UPDATE steps SET finished_at=?", old)
	if err := e.maintain(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	var n int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&n)
	if n != 1 {
		t.Fatal("removed pending event or retained expired completed event", n)
	}
	result, err := e.ingest(ctx, Event{ID: "completed", Source: "local:test"})
	if err != nil || result.(map[string]any)["duplicate"] != true {
		t.Fatal("lost dedup tombstone", result, err)
	}
	raw, _ := json.Marshal(StoragePolicy{100, 1 << 20, 1, 2})
	if _, err = e.setStoragePolicy(ctx, raw); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		_, err = e.ingest(ctx, Event{Source: "local:test", Data: map[string]any{"blob": strings.Repeat("x", 200000)}})
		if i < 5 && err != nil {
			t.Fatal(err)
		}
		if i == 5 && err == nil {
			t.Fatal("payload quota not enforced")
		}
	}
}

func TestSQLiteFullDoesNotAcknowledgeOrKeepPartialEvent(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	var pages int
	e.db.QueryRow("PRAGMA page_count").Scan(&pages)
	if _, err := e.db.Exec(fmt.Sprintf("PRAGMA max_page_count=%d", pages)); err != nil {
		t.Fatal(err)
	}
	_, err := e.ingest(ctx, Event{ID: "full-event", Source: "local:test", Data: map[string]any{"large": strings.Repeat("x", 200000)}})
	var databaseError *sqlite.Error
	if !errors.As(err, &databaseError) || databaseError.Code()&255 != 13 {
		t.Fatal("expected actual SQLITE_FULL", err)
	}
	var n int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&n)
	if n != 0 {
		t.Fatal("partial event persisted")
	}
	for _, table := range []string{"executions", "seen_deliveries"} {
		if err := e.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal("partial ingress transaction", table, n, err)
		}
	}
	if _, err := e.db.Exec("PRAGMA max_page_count=100000"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ingest(ctx, Event{ID: "full-event", Source: "local:test", Data: map[string]any{"large": strings.Repeat("x", 200000)}}); err != nil {
		t.Fatal("storage recovery rejected retry", err)
	}
	if err := e.db.QueryRow("SELECT count(*) FROM executions").Scan(&n); err != nil || n != 1 {
		t.Fatal("retry not admitted exactly once", n, err)
	}
}
func TestMigrationPreservesRowsAndRejectsFutureSchema(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	e.ingest(ctx, Event{ID: "before-migration", Source: "local:test"})
	e.db.Exec("ALTER TABLE executions DROP COLUMN context;DROP TABLE seen_deliveries;DELETE FROM schema_version;INSERT INTO schema_version VALUES(1)")
	if err := migrate(ctx, e.db); err != nil {
		t.Fatal(err)
	}
	var n int
	e.db.QueryRow("SELECT count(*) FROM seen_deliveries WHERE id='before-migration'").Scan(&n)
	if n != 1 {
		t.Fatal("migration did not preserve dedup")
	}
	e.db.Exec("DELETE FROM schema_version;INSERT INTO schema_version VALUES(999)")
	if err := migrate(ctx, e.db); err == nil {
		t.Fatal("accepted future schema")
	}
	e.db.QueryRow("SELECT version FROM schema_version").Scan(&n)
	if n != 999 {
		t.Fatal("modified future schema")
	}
}

func TestJournalMigrationPreservesTimerAndEvent(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, timerConfig("coalesce"))
	if err := e.scheduleTimers(ctx, time.Unix(1800000000, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ingest(ctx, Event{ID: "preserved", Source: "local:test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec("ALTER TABLE executions DROP COLUMN context;DROP TABLE journal_state;DELETE FROM schema_version;INSERT INTO schema_version VALUES(3)"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, e.db); err != nil {
		t.Fatal(err)
	}
	var next int64
	if err := e.db.QueryRow("SELECT next_at FROM timer_state WHERE id='backup'").Scan(&next); err != nil || next != 1800000010 {
		t.Fatal(next, err)
	}
	var count int
	if err := e.db.QueryRow("SELECT count(*) FROM events WHERE id='preserved'").Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	if err := e.db.QueryRow("SELECT count(*) FROM journal_state").Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
	if err := e.db.QueryRow("SELECT version FROM schema_version").Scan(&count); err != nil || count != DatabaseVersion {
		t.Fatal(count, err)
	}
}
