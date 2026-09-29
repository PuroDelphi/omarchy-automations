package core

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"quatrro.local/automations/internal/adapters"
	"strings"
	"time"
)

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func (e *Engine) ingest(ctx context.Context, ev Event) (any, error) {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	c, err := e.config(ctx, "active")
	if err != nil {
		return nil, err
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := e.enqueue(ctx, tx, c, ev)
	if err != nil {
		return nil, err
	}
	return result, tx.Commit()
}

// enqueue shares its transaction with monitor/scheduler state transitions.
func (e *Engine) enqueue(ctx context.Context, tx *sql.Tx, c Config, ev Event) (any, error) {
	if ev.Source == "" || len(ev.Source) > 128 || len(ev.Type) > 128 || ev.Depth < 0 || ev.Depth > 8 {
		return nil, errors.New("invalid event envelope")
	}
	if ev.ID == "" {
		ev.ID = newID()
	}
	if len(ev.ID) > 200 {
		return nil, errors.New("event ID too long")
	}
	if ev.Correlation == "" {
		ev.Correlation = ev.ID
	}
	raw, err := json.Marshal(ev)
	if err != nil || len(raw) > 262144 {
		return nil, errors.New("event exceeds 256 KiB")
	}
	var paused string
	_ = tx.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='admission'").Scan(&paused)
	if paused == "reject" {
		return nil, errors.New("event admission paused")
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM events").Scan(&n); err != nil {
		return nil, err
	}
	var duplicate int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM seen_deliveries WHERE id=? AND source=?", ev.ID, ev.Source).Scan(&duplicate); err != nil {
		return nil, err
	}
	if duplicate > 0 {
		return map[string]any{"id": ev.ID, "duplicate": true}, nil
	}
	policy, err := storagePolicy(ctx, tx)
	if err != nil {
		return nil, err
	}
	if n >= policy.MaxEvents {
		return nil, errors.New("event capacity reached")
	}
	var used, seen int64
	if err = tx.QueryRowContext(ctx, payloadUsageSQL).Scan(&used); err != nil {
		return nil, err
	}
	if used+int64(len(raw)) > policy.MaxPayloadBytes {
		return nil, errors.New("payload storage quota reached")
	}
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM seen_deliveries").Scan(&seen); err != nil {
		return nil, err
	}
	if seen >= 1000000 {
		return nil, errors.New("deduplication capacity reached")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO events(id,source,payload,created_at) VALUES(?,?,?,?)", ev.ID, ev.Source, string(raw), now)
	if err != nil {
		return nil, err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if inserted == 0 {
		return map[string]any{"id": ev.ID, "duplicate": true}, nil
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO seen_deliveries(id,source,seen_at) VALUES(?,?,?)", ev.ID, ev.Source, time.Now().Unix()); err != nil {
		return nil, err
	}
	revision := canonicalHash(c)
	b, _ := json.Marshal(c)
	if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO revisions(hash,config,created_at) VALUES(?,?,?)", revision, string(b), now); err != nil {
		return nil, err
	}
	count := 0
	for _, f := range c.Flows {
		if matches(f, ev) {
			id := newID()
			if _, err = tx.ExecContext(ctx, "INSERT INTO executions(id,event_id,event_source,revision,flow,state,step,created_at) VALUES(?,?,?,?,?,'pending',0,?)", id, ev.ID, ev.Source, revision, f.ID, now); err != nil {
				return nil, err
			}
			count++
		}
	}
	return map[string]any{"id": ev.ID, "executions": count, "duplicate": false}, nil
}
func (e *Engine) emit(ctx context.Context, data []byte) (any, error) {
	var ev Event
	if err := decode(data, &ev); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(ev.Source, "local:") && !strings.HasPrefix(ev.Source, "hook:") {
		return nil, errors.New("local emitter accepts only local: or hook: sources")
	}
	return e.ingest(ctx, ev)
}
func (e *Engine) simulate(ctx context.Context, data []byte) (any, error) {
	var ev Event
	if err := decode(data, &ev); err != nil {
		return nil, err
	}
	c, err := e.config(ctx, "draft")
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	evaluations := []map[string]any{}
	for _, f := range c.Flows {
		conditions := []map[string]any{}
		for _, condition := range f.Conditions {
			conditions = append(conditions, map[string]any{"field": condition.Field, "operator": condition.Op, "expected": condition.Value, "matched": conditionMatches(condition, ev)})
		}
		evaluations = append(evaluations, map[string]any{"flow": f.ID, "enabled": f.Enabled, "source_matched": f.Source == ev.Source, "matched": matches(f, ev), "conditions": conditions})
		if !matches(f, ev) {
			continue
		}
		steps := []map[string]any{}
		deferred := false
		for _, id := range f.Steps {
			for _, a := range c.Actions {
				if a.ID != id {
					continue
				}
				if deferred {
					steps = append(steps, map[string]any{"action": id, "kind": a.Kind, "requires_adapter_result": true})
					continue
				}
				if a.Kind == "adapter" {
					adapter, err := selectedAdapter(c, a)
					if err != nil {
						return nil, err
					}
					if _, err := adapters.EncodeRequest(adapter.Manifest, "simulation", adapters.Event{Source: ev.Source, Type: ev.Type, Data: ev.Data}); err != nil {
						return nil, err
					}
					steps = append(steps, map[string]any{"action": id, "kind": a.Kind, "adapter": adapter.ID, "adapter_revision": adapter.Revision, "result_available": false})
					deferred = true
					continue
				}
				title, err := render(a.Title, ev)
				if err != nil {
					return nil, err
				}
				body, err := render(a.Body, ev)
				if a.Kind == "http" {
					var raw []byte
					raw, err = renderHTTP(c, a, ev)
					body = string(raw)
				}
				if err != nil {
					return nil, err
				}
				step := map[string]any{"action": id, "kind": a.Kind, "title": title, "body": body, "resource": a.Unit, "destination": a.Destination}
				if a.Kind == "system-service" {
					step["operation"] = a.Operation
					step["administrative_broker_required"] = true
				}
				if a.Kind == "command" {
					resolved, err := resolveCommand(a)
					if err != nil {
						return nil, err
					}
					step["command_profile"] = a.CommandProfile
					step["executable"] = resolved.Executable
					step["arguments"] = resolved.Args
				}
				if a.Kind == "command" || a.Kind == "script" {
					step["directories"] = a.Directories
					step["working_directory"] = a.WorkingDirectory
				}
				if a.Kind == "script" {
					args, err := resolveScriptArguments(c, a, ev)
					if err != nil {
						return nil, err
					}
					step["script"] = a.Script
					step["script_revision"] = a.ScriptRevision
					step["arguments"] = args
				}
				steps = append(steps, step)
			}
		}
		out = append(out, map[string]any{"flow": f.ID, "steps": steps})
	}
	return map[string]any{"matches": out, "evaluations": evaluations, "effects_executed": false, "capabilities": capabilities(c)}, nil
}
