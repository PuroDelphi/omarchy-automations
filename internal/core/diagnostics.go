package core

import (
	"context"
	"database/sql"
	"errors"
	"os/exec"
	"runtime"
	"time"

	"quatrro.local/automations/internal/local"
)

// This report is built from an allowlist, never by redacting a config dump.
// No identifiers, payloads, URLs, paths, environment, secret values or raw errors.
func (e *Engine) diagnostics(ctx context.Context) (any, error) {
	status, err := e.status(ctx)
	if err != nil {
		return nil, err
	}
	storage, err := e.storageStatus(ctx)
	if err != nil {
		return nil, err
	}
	active, err := e.config(ctx, "active")
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for label, query := range map[string]string{
		"grants": "SELECT count(*) FROM grants", "revisions": "SELECT count(*) FROM revisions",
		"secret_references": "SELECT count(*) FROM secrets", "outbox_pending": "SELECT count(*) FROM outbox WHERE state='pending'",
		"outbox_delivered": "SELECT count(*) FROM outbox WHERE state='delivered'", "monitor_errors": "SELECT (SELECT count(*) FROM monitor_state WHERE error!='')+(SELECT count(*) FROM journal_state WHERE error!='')",
	} {
		var n int
		if err := e.db.QueryRowContext(ctx, query).Scan(&n); err != nil {
			return nil, err
		}
		counts[label] = n
	}
	var databaseVersion int
	var sqliteVersion string
	if err = e.db.QueryRowContext(ctx, "SELECT max(version) FROM schema_version").Scan(&databaseVersion); err != nil {
		return nil, err
	}
	if err = e.db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&sqliteVersion); err != nil {
		return nil, err
	}
	var admission string
	if err = e.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='admission'").Scan(&admission); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	state := status.(map[string]any)
	knownError := ""
	switch state["last_error"] {
	case "worker storage failure", "monitor storage failure", "storage maintenance failed", "scheduler storage failure":
		knownError = state["last_error"].(string)
	}
	executionCounts := map[string]int{}
	for _, name := range []string{"pending", "running", "completed", "failed", "cancelled", "denied", "uncertain"} {
		executionCounts[name] = state["counts"].(map[string]int)[name]
	}
	tools := map[string]bool{}
	for _, name := range []string{"systemctl", "systemd-run", "bwrap", "notify-send", "secret-tool", "omarchy"} {
		_, err := exec.LookPath(name)
		tools[name] = err == nil
	}
	return map[string]any{
		"report_version": 1, "generated_at": time.Now().UTC().Format(time.RFC3339),
		"versions":         map[string]any{"engine": Version, "protocol": local.Version, "database": databaseVersion, "sqlite": sqliteVersion, "go": runtime.Version()},
		"platform":         map[string]string{"os": runtime.GOOS, "architecture": runtime.GOARCH},
		"operation":        map[string]any{"paused": state["paused"], "admission_rejected": admission == "reject", "uptime_seconds": state["uptime_seconds"], "last_error_code": knownError},
		"execution_counts": executionCounts, "counts": counts, "storage": storage,
		"active_resources": map[string]int{"entries": len(active.Entries), "destinations": len(active.Destinations), "actions": len(active.Actions), "flows": len(active.Flows), "monitors": len(active.Monitors), "timers": len(active.Timers)},
		"tools_on_path":    tools,
	}, nil
}

func (e *Engine) diagnosticsFile(ctx context.Context, data []byte) (any, error) {
	path, err := exportPath(data)
	if err != nil {
		return nil, err
	}
	report, err := e.diagnostics(ctx)
	if err != nil {
		return nil, err
	}
	return writeJSONExport(path, report)
}

func (e *Engine) queueInspect(ctx context.Context, data []byte) (any, error) {
	var req struct {
		State  string `json:"state"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if len(data) > 0 {
		if err := decode(data, &req); err != nil {
			return nil, err
		}
	}
	switch req.State {
	case "", "pending", "running", "uncertain":
	default:
		return nil, errors.New("queue state must be pending, running or uncertain")
	}
	if req.Limit == 0 {
		req.Limit = 100
	}
	if req.Limit < 1 || req.Limit > 100 || req.Offset < 0 || req.Offset > 100000 {
		return nil, errors.New("queue page outside limits")
	}
	filter := `x.state IN ('pending','running','uncertain') AND (?='' OR x.state=?)`
	var total int
	if err := e.db.QueryRowContext(ctx, "SELECT count(*) FROM executions x WHERE "+filter, req.State, req.State).Scan(&total); err != nil {
		return nil, err
	}
	rows, err := e.db.QueryContext(ctx, `SELECT x.id,x.flow,x.state,x.step,x.created_at,COALESCE(o.attempts,0),COALESCE(o.next_at,0) FROM executions x LEFT JOIN outbox o ON o.execution=x.id AND o.step=x.step WHERE `+filter+` ORDER BY x.created_at,x.id LIMIT ? OFFSET ?`, req.State, req.State, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, flow, state, created string
		var step, attempts int
		var next int64
		if err := rows.Scan(&id, &flow, &state, &step, &created, &attempts, &next); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "flow": flow, "state": state, "step": step, "created_at": created, "attempts": attempts, "next_at": next})
	}
	// Marshal is deliberately not used on executions or events: only these fields.
	return map[string]any{"items": items, "total": total, "offset": req.Offset, "limit": req.Limit, "has_more": req.Offset+len(items) < total}, rows.Err()
}
