package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Include per-execution transformed contexts and count UTF-8 bytes, not runes.
const payloadUsageSQL = "SELECT (SELECT COALESCE(sum(length(CAST(payload AS BLOB))),0) FROM events) + (SELECT COALESCE(sum(length(CAST(context AS BLOB))),0) FROM executions)"

type StoragePolicy struct {
	MaxEvents       int   `json:"max_events"`
	MaxPayloadBytes int64 `json:"max_payload_bytes"`
	RetentionDays   int   `json:"retention_days"`
	DedupDays       int   `json:"dedup_days"`
}

func defaultStoragePolicy() StoragePolicy { return StoragePolicy{10000, 64 << 20, 7, 30} }

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func storagePolicy(ctx context.Context, q queryRower) (StoragePolicy, error) {
	p := defaultStoragePolicy()
	var raw string
	err := q.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='storage_policy'").Scan(&raw)
	if err == sql.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	err = json.Unmarshal([]byte(raw), &p)
	return p, err
}
func (e *Engine) setStoragePolicy(ctx context.Context, data []byte) (any, error) {
	var p StoragePolicy
	if err := decode(data, &p); err != nil {
		return nil, err
	}
	if p.MaxEvents < 100 || p.MaxEvents > 100000 || p.MaxPayloadBytes < 1<<20 || p.MaxPayloadBytes > 128<<20 || p.RetentionDays < 1 || p.RetentionDays > 90 || p.DedupDays < p.RetentionDays || p.DedupDays > 365 {
		return nil, errors.New("storage bounds: 100..100000 events, 1..128 MiB payloads, 1..90 retention days, dedup between retention and 365 days")
	}
	raw, _ := json.Marshal(p)
	e.mutation.Lock()
	defer e.mutation.Unlock()
	_, err := e.db.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES('storage_policy',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", string(raw))
	return p, err
}
func (e *Engine) storageStatus(ctx context.Context) (any, error) {
	p, err := storagePolicy(ctx, e.db)
	if err != nil {
		return nil, err
	}
	var count, payload, dedup, pages, pageSize int64
	if err = e.db.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM events), ("+payloadUsageSQL+")").Scan(&count, &payload); err != nil {
		return nil, err
	}
	if err = e.db.QueryRowContext(ctx, "SELECT count(*) FROM seen_deliveries").Scan(&dedup); err != nil {
		return nil, err
	}
	if err = e.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); err != nil {
		return nil, err
	}
	if err = e.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return nil, err
	}
	return map[string]any{"policy": p, "events": count, "payload_bytes": payload, "dedup_records": dedup, "database_bytes": pages * pageSize, "database_limit_bytes": 65536 * pageSize}, nil
}
func (e *Engine) RunMaintenance(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := e.maintain(ctx, time.Now()); err != nil && ctx.Err() == nil {
			e.lastError.Store("storage maintenance failed")
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (e *Engine) maintain(ctx context.Context, now time.Time) error {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	p, err := storagePolicy(ctx, tx)
	if err != nil {
		return err
	}
	cutoff := now.Add(-time.Duration(p.RetentionDays) * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	// Uncertain effects are kept for manual inspection. Never evict runnable jobs.
	ids := `SELECT x.id FROM executions x WHERE x.state IN ('completed','failed','cancelled','denied') AND COALESCE((SELECT MAX(finished_at) FROM steps WHERE execution=x.id),x.created_at)<?`
	if _, err = tx.ExecContext(ctx, "DELETE FROM outbox WHERE execution IN ("+ids+")", cutoff); err != nil {
		return err
	}
	// Materialize IDs before deleting step timestamps used by the cutoff.
	if _, err = tx.ExecContext(ctx, "CREATE TEMP TABLE IF NOT EXISTS cleanup_ids(id TEXT PRIMARY KEY);DELETE FROM cleanup_ids;"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO cleanup_ids "+ids, cutoff); err != nil {
		return err
	}
	for _, statement := range []string{"DELETE FROM steps WHERE execution IN (SELECT id FROM cleanup_ids)", "DELETE FROM executions WHERE id IN (SELECT id FROM cleanup_ids)"} {
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM events WHERE created_at<? AND NOT EXISTS(SELECT 1 FROM executions WHERE event_id=events.id AND event_source=events.source)", cutoff); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM seen_deliveries WHERE seen_at<? AND NOT EXISTS(SELECT 1 FROM events WHERE id=seen_deliveries.id AND source=seen_deliveries.source)", now.Add(-time.Duration(p.DedupDays)*24*time.Hour).Unix()); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM audit WHERE at<?", cutoff); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM revisions WHERE created_at<? AND NOT EXISTS(SELECT 1 FROM executions WHERE revision=revisions.hash) AND config NOT IN (SELECT value FROM settings WHERE key IN ('active','draft'))", cutoff); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	_, err = e.db.ExecContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)")
	return err
}
