package core

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

func canonicalHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func (e *Engine) config(ctx context.Context, key string) (Config, error) {
	var raw string
	err := e.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key=?", key).Scan(&raw)
	if err == sql.ErrNoRows {
		return EmptyConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	err = json.Unmarshal([]byte(raw), &c)
	return c, err
}
func (e *Engine) saveDraft(ctx context.Context, data []byte) (any, error) {
	var c Config
	if err := decode(data, &c); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(c)
	e.mutation.Lock()
	defer e.mutation.Unlock()
	_, err := e.db.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES('draft',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", string(b))
	return map[string]any{"hash": canonicalHash(c)}, err
}

// Capabilities bind the complete action and its destination to a revision hash.
// Changing resource configuration cannot reuse an earlier authorization.
func actionScope(flow, action string) string {
	return "flow:" + flow + ":action:" + action
}

func capabilities(c Config) map[string]string {
	out := map[string]string{}
	for _, m := range c.Monitors {
		if m.Enabled {
			out["monitor:"+m.ID] = monitorHash(c, m)
		}
	}
	for _, t := range c.Timers {
		if t.Enabled {
			out["timer:"+t.ID] = canonicalHash(t)
		}
	}
	actions := map[string]Action{}
	for _, a := range c.Actions {
		actions[a.ID] = a
	}
	for _, f := range c.Flows {
		if !f.Enabled {
			continue
		}
		for _, id := range f.Steps {
			a := actions[id]
			var dest *Destination
			for i := range c.Destinations {
				if c.Destinations[i].ID == a.Destination {
					dest = &c.Destinations[i]
					break
				}
			}
			scope := struct {
				Flow        Flow
				Action      Action
				Destination *Destination
			}{f, a, dest}
			out[actionScope(f.ID, id)] = canonicalHash(scope)
		}
	}
	return out
}
func (e *Engine) activate(ctx context.Context, data []byte) (any, error) {
	var req struct {
		Hash   string            `json:"hash"`
		Grants map[string]string `json:"grants"`
	}
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	c, err := e.config(ctx, "draft")
	if err != nil {
		return nil, err
	}
	hash := canonicalHash(c)
	if hash != req.Hash {
		return nil, errors.New("draft changed; preview again")
	}
	required := capabilities(c)
	if len(required) != len(req.Grants) {
		return nil, errors.New("explicit grant required for each capability")
	}
	for k, v := range required {
		if req.Grants[k] != v {
			return nil, fmt.Errorf("missing capability grant: %s", k)
		}
	}
	if err = e.preflight(ctx, c); err != nil {
		return nil, err
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	b, _ := json.Marshal(c)
	if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO revisions(hash,config,created_at) VALUES(?,?,?)", hash, string(b), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES('active',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", string(b)); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM grants"); err != nil {
		return nil, err
	}
	for k, v := range required {
		if _, err = tx.ExecContext(ctx, "INSERT INTO grants(scope,hash) VALUES(?,?)", k, v); err != nil {
			return nil, err
		}
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO audit(at,operation,resource,result) VALUES(?,?,?,?)", time.Now().UTC().Format(time.RFC3339Nano), "activate", hash, "ok")
	if err != nil {
		return nil, err
	}
	return map[string]string{"revision": hash}, tx.Commit()
}
func (e *Engine) preview(ctx context.Context) (any, error) {
	c, err := e.config(ctx, "draft")
	if err != nil {
		return nil, err
	}
	return map[string]any{"hash": canonicalHash(c), "capabilities": capabilities(c), "config": c}, nil
}
func (e *Engine) revoke(ctx context.Context, data []byte) (any, error) {
	var req struct {
		Scope string `json:"scope"`
	}
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	_, err := e.db.ExecContext(ctx, "DELETE FROM grants WHERE scope=?", req.Scope)
	return map[string]string{"revoked": req.Scope}, err
}
