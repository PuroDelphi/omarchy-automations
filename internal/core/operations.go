package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"quatrro.local/automations/internal/local"
)

func (e *Engine) status(ctx context.Context) (any, error) {
	var paused string
	err := e.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='paused'").Scan(&paused)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	rows, err := e.db.QueryContext(ctx, "SELECT state,count(*) FROM executions GROUP BY state")
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for rows.Next() {
		var state string
		var n int
		if err := rows.Scan(&state, &n); err != nil {
			rows.Close()
			return nil, err
		}
		counts[state] = n
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	state := "ready"
	if paused == "true" {
		state = "paused"
	}
	last := ""
	if value := e.lastError.Load(); value != nil {
		last = value.(string)
	}
	preferences, err := e.preferences(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"version": Version, "protocol": local.Version, "uptime_seconds": int(time.Since(e.started).Seconds()), "state": state, "paused": paused == "true", "counts": counts, "ingress": e.ingressStatus(), "last_error": last, "language": preferences.(map[string]string)["language"]}, nil
}
func (e *Engine) permissions(ctx context.Context) (any, error) {
	rows, err := e.db.QueryContext(ctx, "SELECT scope,hash FROM grants ORDER BY scope")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var scope, hash string
		if err := rows.Scan(&scope, &hash); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{"scope": scope, "hash": hash})
	}
	return out, rows.Err()
}
func (e *Engine) detail(ctx context.Context, data []byte) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	rows, err := e.db.QueryContext(ctx, "SELECT step,state,message,finished_at FROM steps WHERE execution=? ORDER BY step", req.ID)
	if err != nil {
		return nil, err
	}
	steps := []map[string]any{}
	for rows.Next() {
		var step int
		var state, message, at string
		if err := rows.Scan(&step, &state, &message, &at); err != nil {
			rows.Close()
			return nil, err
		}
		// Earlier releases recorded the execution's next state for successful steps.
		// A persisted step marked pending was finished, not awaiting delivery.
		if state == "pending" {
			state = "completed"
		}
		steps = append(steps, map[string]any{"step": step, "state": state, "message": message, "at": at})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = e.db.QueryContext(ctx, "SELECT step,state,attempts,next_at,last_status FROM outbox WHERE execution=? ORDER BY step", req.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deliveries := []map[string]any{}
	for rows.Next() {
		var step, attempts, next int
		var status sql.NullInt64
		var state string
		if err := rows.Scan(&step, &state, &attempts, &next, &status); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, map[string]any{"step": step, "state": state, "attempts": attempts, "next_at": next, "status": status.Int64})
	}
	return map[string]any{"id": req.ID, "steps": steps, "deliveries": deliveries}, rows.Err()
}
func (e *Engine) retry(ctx context.Context, data []byte) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var step int
	var state string
	if err = tx.QueryRowContext(ctx, "SELECT step,state FROM executions WHERE id=?", req.ID).Scan(&step, &state); err != nil {
		return nil, errors.New("execution unavailable")
	}
	if state != "failed" && state != "cancelled" {
		return nil, errors.New("only failed or cancelled HTTP deliveries can be retried")
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM outbox WHERE execution=? AND step=? AND state!='delivered'", req.ID, step).Scan(&n); err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, errors.New("manual retry is limited to HTTP deliveries; inspect non-idempotent actions")
	}
	if _, err = tx.ExecContext(ctx, "UPDATE outbox SET attempts=0,next_at=0,created_at=?,state='pending' WHERE execution=? AND step=?", time.Now().Unix(), req.ID, step); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE executions SET state='pending' WHERE id=?", req.ID); err != nil {
		return nil, err
	}
	return map[string]string{"id": req.ID, "state": "pending"}, tx.Commit()
}
func (e *Engine) importConfig(ctx context.Context, data []byte) (any, error) {
	var c Config
	if err := decode(data, &c); err != nil {
		return nil, err
	}
	for i := range c.Entries {
		c.Entries[i].Enabled = false
	}
	for i := range c.Flows {
		c.Flows[i].Enabled = false
	}
	for i := range c.Monitors {
		c.Monitors[i].Enabled = false
	}
	for i := range c.Timers {
		c.Timers[i].Enabled = false
	}
	raw, _ := json.Marshal(c)
	return e.saveDraft(ctx, raw)
}
func (e *Engine) configFile(ctx context.Context, data []byte, write bool) (any, error) {
	path, err := exportPath(data)
	if err != nil {
		return nil, err
	}
	if write {
		c, err := e.config(ctx, "draft")
		if err != nil {
			return nil, err
		}
		return writeJSONExport(path, c)
	}
	raw, err := readJSONImport(path)
	if err != nil {
		return nil, err
	}
	return e.importConfig(ctx, raw)
}
