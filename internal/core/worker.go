package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"quatrro.local/automations/internal/broker"
	"time"
)

type job struct {
	ID, Revision, Flow, State string
	Step                      int
	Event                     Event
	Config                    Config
}

var errExecutionContextQuota = errors.New("adapter execution context exceeds storage quota")

var errUserCancelled = errors.New("user cancelled execution")

func (e *Engine) Run(ctx context.Context) error {
	// Never silently repeat a non-idempotent effect whose result was not persisted.
	if _, err := e.db.ExecContext(ctx, "UPDATE executions SET state=CASE WHEN EXISTS(SELECT 1 FROM outbox WHERE execution=executions.id AND step=executions.step) THEN 'pending' ELSE 'uncertain' END WHERE state='running'"); err != nil {
		return err
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := e.tick(ctx); err != nil && ctx.Err() == nil {
				e.lastError.Store("worker storage failure")
			}
		}
	}
}
func (e *Engine) tick(ctx context.Context) error {
	e.mutation.Lock()
	var paused string
	_ = e.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='paused'").Scan(&paused)
	if paused == "true" {
		e.mutation.Unlock()
		return nil
	}
	var j job
	var ev, config string
	err := e.db.QueryRowContext(ctx, `SELECT x.id,x.revision,x.flow,x.state,x.step,COALESCE(NULLIF(x.context,''),v.payload),r.config FROM executions x JOIN events v ON x.event_id=v.id AND x.event_source=v.source JOIN revisions r ON x.revision=r.hash WHERE x.state='pending' AND NOT EXISTS(SELECT 1 FROM outbox o WHERE o.execution=x.id AND o.step=x.step AND o.next_at>?) ORDER BY x.created_at,x.id LIMIT 1`, time.Now().Unix()).Scan(&j.ID, &j.Revision, &j.Flow, &j.State, &j.Step, &ev, &config)
	if err == sql.ErrNoRows {
		e.mutation.Unlock()
		return nil
	}
	if err != nil {
		e.mutation.Unlock()
		return err
	}
	if err = json.Unmarshal([]byte(ev), &j.Event); err != nil {
		e.mutation.Unlock()
		return err
	}
	if err = json.Unmarshal([]byte(config), &j.Config); err != nil {
		e.mutation.Unlock()
		return err
	}
	var f Flow
	for _, v := range j.Config.Flows {
		if v.ID == j.Flow {
			f = v
			break
		}
	}
	if j.Step >= len(f.Steps) {
		_, err = e.db.ExecContext(ctx, "UPDATE executions SET state='completed' WHERE id=?", j.ID)
		e.mutation.Unlock()
		return err
	}
	var a Action
	for _, v := range j.Config.Actions {
		if v.ID == f.Steps[j.Step] {
			a = v
			break
		}
	}
	var grant string
	err = e.db.QueryRowContext(ctx, "SELECT hash FROM grants WHERE scope=?", actionScope(f.ID, a.ID)).Scan(&grant)
	if err != nil || grant != capabilities(j.Config)[actionScope(f.ID, a.ID)] {
		err = e.finishJob(ctx, j, "denied", "action permission missing or changed")
		e.mutation.Unlock()
		return err
	}
	if a.Kind == "http" {
		body, err := renderHTTP(j.Config, a, j.Event)
		if err != nil {
			e.mutation.Unlock()
			return e.failJob(ctx, j, "HTTP action template invalid")
		}
		// Persist the exact bytes once; retries never re-evaluate templates.
		_, err = e.db.ExecContext(ctx, "INSERT OR IGNORE INTO outbox(execution,step,body,attempts,next_at,created_at,state) VALUES(?,?,?,0,0,?,'pending')", j.ID, j.Step, string(body), time.Now().Unix())
		if err != nil {
			e.mutation.Unlock()
			return err
		}
	}
	if _, err = e.db.ExecContext(ctx, "UPDATE executions SET state='running' WHERE id=?", j.ID); err != nil {
		e.mutation.Unlock()
		return err
	}
	workCtx, cancel := context.WithCancelCause(ctx)
	e.runningMu.Lock()
	e.running[j.ID] = func() { cancel(errUserCancelled) }
	e.runningMu.Unlock()
	e.mutation.Unlock()
	defer func() { cancel(nil); e.runningMu.Lock(); delete(e.running, j.ID); e.runningMu.Unlock() }()
	if a.Kind == "http" {
		return e.deliver(workCtx, j, a)
	}
	var adapted map[string]any
	var administrative broker.Result
	if a.Kind == "system-service" {
		administrative, err = e.performAdministrative(workCtx, a, j.ID)
	} else if a.Kind == "adapter" {
		var adapter Adapter
		adapter, err = selectedAdapter(j.Config, a)
		if err == nil {
			adapted, err = executeAdapter(workCtx, adapter, j.Event)
		}
	} else if a.Kind == "script" {
		err = e.performScript(workCtx, j.Config, a, j.Event)
	} else {
		err = e.perform(workCtx, a, j.Event)
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	if workCtx.Err() != nil {
		return e.finishJob(context.WithoutCancel(ctx), j, "uncertain", "cancelled after dispatch; inspect effect")
	}
	if err != nil {
		if a.Kind == "system-service" && errors.Is(err, broker.ErrOutcomeUnknown) {
			return e.finishJob(ctx, j, "uncertain", err.Error())
		}
		if a.Kind == "system-service" && errors.Is(err, broker.ErrDenied) {
			return e.finishJob(ctx, j, "denied", err.Error())
		}
		return e.finishJob(ctx, j, "failed", err.Error())
	}
	if a.Kind == "adapter" {
		err := e.finishAdapter(ctx, j, adapted)
		if errors.Is(err, errExecutionContextQuota) {
			return e.finishJob(ctx, j, "failed", err.Error())
		}
		return err
	}
	if a.Kind == "system-service" && a.Operation == "status" {
		return e.finishJob(ctx, j, "pending", administrative.LoadState+" / "+administrative.ActiveState+" / "+administrative.SubState)
	}
	return e.finishJob(ctx, j, "pending", "ok")
}
func (e *Engine) failJob(ctx context.Context, j job, message string) error {
	return e.finishJob(ctx, j, "failed", message)
}
func (e *Engine) finishJob(ctx context.Context, j job, state, message string) error {
	return e.finishJobContext(ctx, j, state, message, nil)
}
func (e *Engine) finishJobContext(ctx context.Context, j job, state, message string, updated *Event) error {
	if len(message) > 200 {
		message = message[:200]
	}
	step := j.Step
	stepState := state
	if state == "pending" {
		step++
		stepState = "completed"
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO steps(execution,step,state,message,finished_at) VALUES(?,?,?,?,?) ON CONFLICT(execution,step) DO UPDATE SET state=excluded.state,message=excluded.message,finished_at=excluded.finished_at", j.ID, j.Step, stepState, message, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if updated != nil {
		raw, err := json.Marshal(updated)
		if err != nil {
			return err
		}
		if len(raw) > 262144 {
			return errExecutionContextQuota
		}
		policy, err := storagePolicy(ctx, tx)
		if err != nil {
			return err
		}
		var used, previous int64
		if err = tx.QueryRowContext(ctx, payloadUsageSQL).Scan(&used); err != nil {
			return err
		}
		if err = tx.QueryRowContext(ctx, "SELECT length(CAST(context AS BLOB)) FROM executions WHERE id=?", j.ID).Scan(&previous); err != nil {
			return err
		}
		if used-previous+int64(len(raw)) > policy.MaxPayloadBytes {
			return errExecutionContextQuota
		}
		if _, err = tx.ExecContext(ctx, "UPDATE executions SET state=?,step=?,context=? WHERE id=?", state, step, string(raw), j.ID); err != nil {
			return err
		}
	} else if _, err = tx.ExecContext(ctx, "UPDATE executions SET state=?,step=? WHERE id=?", state, step, j.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (e *Engine) deliver(ctx context.Context, j job, a Action) error {
	var d Destination
	for _, v := range j.Config.Destinations {
		if v.ID == a.Destination {
			d = v
			break
		}
	}
	var body, previous string
	var attempts, created int64
	if err := e.db.QueryRowContext(ctx, "SELECT body,attempts,created_at,state FROM outbox WHERE execution=? AND step=?", j.ID, j.Step).Scan(&body, &attempts, &created, &previous); err != nil {
		return err
	}
	switch previous {
	case "delivered":
		return e.finishJob(ctx, j, "pending", "delivery already persisted")
	case "failed":
		return e.finishJob(ctx, j, "failed", "delivery failure already persisted")
	case "uncertain":
		return e.finishJob(ctx, j, "uncertain", "delivery outcome uncertain; inspect effect")
	}
	if attempts >= 8 || time.Now().Unix()-created >= 86400 {
		return e.finishJob(ctx, j, "failed", "delivery expired")
	}
	// Reserve the attempt durably before dispatch. A crash after an external
	// effect must not reset the delivery budget on every restart.
	attempts++
	if _, err := e.db.ExecContext(ctx, "UPDATE outbox SET attempts=? WHERE execution=? AND step=?", attempts, j.ID, j.Step); err != nil {
		return err
	}
	result := e.send(ctx, d, fmt.Sprintf("%s:%d", j.ID, j.Step), []byte(body))
	e.mutation.Lock()
	defer e.mutation.Unlock()
	if errors.Is(context.Cause(ctx), errUserCancelled) {
		if _, err := e.db.ExecContext(context.WithoutCancel(ctx), "UPDATE outbox SET state='uncertain' WHERE execution=? AND step=?", j.ID, j.Step); err != nil {
			return err
		}
		return e.finishJob(context.WithoutCancel(ctx), j, "uncertain", "HTTP cancelled after dispatch; delivery may have occurred")
	}
	next := int64(0)
	state := "failed"
	if result.Status >= 200 && result.Status < 300 {
		state = "delivered"
	} else if result.Retry && attempts < 8 && time.Now().Unix()-created < 86400 {
		state = "pending"
		delay := time.Second*time.Duration(1<<min(attempts, 10)) + time.Duration(rand.IntN(1000))*time.Millisecond
		if result.After > delay {
			delay = result.After
		}
		next = time.Now().Add(delay).Unix()
	}
	if _, err := e.db.ExecContext(context.WithoutCancel(ctx), "UPDATE outbox SET state=?,attempts=?,next_at=?,last_status=? WHERE execution=? AND step=?", state, attempts, next, result.Status, j.ID, j.Step); err != nil {
		return err
	}
	if state == "pending" {
		_, err := e.db.ExecContext(context.WithoutCancel(ctx), "UPDATE executions SET state='pending' WHERE id=?", j.ID)
		return err
	}
	if state == "delivered" {
		return e.finishJob(context.WithoutCancel(ctx), j, "pending", result.Message)
	}
	return e.finishJob(context.WithoutCancel(ctx), j, "failed", result.Message)
}
func (e *Engine) control(ctx context.Context, data []byte) (any, error) {
	var req struct {
		Paused    bool   `json:"paused"`
		Admission string `json:"admission"`
	}
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	if req.Admission != "retain" && req.Admission != "reject" {
		return nil, errors.New("admission must be retain or reject")
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for k, v := range map[string]string{"paused": fmt.Sprint(req.Paused), "admission": req.Admission} {
		if _, err = tx.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", k, v); err != nil {
			return nil, err
		}
	}
	return map[string]any{"paused": req.Paused, "admission": req.Admission}, tx.Commit()
}
func (e *Engine) cancelJobs(ctx context.Context) (any, error) {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	// Persist cancellation before acknowledging or signalling active workers.
	// Otherwise a crash immediately after a successful cancellation could
	// recover an in-flight HTTP delivery as retryable.
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE outbox SET state='uncertain' WHERE state='pending' AND EXISTS(SELECT 1 FROM executions x WHERE x.id=outbox.execution AND x.step=outbox.step AND x.state='running')`); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE executions SET state='cancelled' WHERE state='pending'"); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	e.runningMu.Lock()
	for _, cancel := range e.running {
		cancel()
	}
	e.runningMu.Unlock()
	return map[string]bool{"cancel_requested": true}, nil
}
func (e *Engine) history(ctx context.Context) (any, error) {
	rows, err := e.db.QueryContext(ctx, "SELECT id,flow,state,step,revision,created_at FROM executions ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, flow, state, revision, created string
		var step int
		if err := rows.Scan(&id, &flow, &state, &step, &revision, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "flow": flow, "state": state, "step": step, "revision": revision, "created_at": created})
	}
	return out, rows.Err()
}
