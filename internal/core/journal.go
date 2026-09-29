package core

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"time"
)

type journalState struct {
	Hash, Cursor, Error string
	PreviousHash        string
	Since, Next         int64
	Count               int
}
type journalRecord struct {
	Cursor, Unit string
	Timestamp    int64
	Priority     int
}

type boundedBuffer struct {
	buffer bytes.Buffer
	Limit  int
}

func (b *boundedBuffer) Bytes() []byte { return b.buffer.Bytes() }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > b.Limit {
		return 0, errors.New("output limit exceeded")
	}
	return b.buffer.Write(p)
}

func readJournal(ctx context.Context, m Monitor, s journalState) ([]journalRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	args := []string{"--user", "--unit=" + m.Unit, "--no-pager", "--quiet", "--output=json", "--output-fields=__CURSOR,__REALTIME_TIMESTAMP,PRIORITY,_SYSTEMD_USER_UNIT", "--priority=" + strconv.Itoa(m.Priority)}
	if s.Cursor != "" {
		probe := append(append([]string{}, args...), "--lines=+1", "--cursor="+s.Cursor)
		records, err := journalCommand(ctx, probe)
		if err != nil || len(records) != 1 || records[0].Cursor != s.Cursor {
			return nil, errors.New("journal cursor is no longer retained")
		}

		args = append(args, "--after-cursor="+s.Cursor)
	} else {
		args = append(args, fmt.Sprintf("--since=@%d.%06d", s.Since/1000000, s.Since%1000000))
	}
	return journalCommand(ctx, append(args, "--lines=+64"))
}

func journalCommand(ctx context.Context, args []string) ([]journalRecord, error) {
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	cmd.Env = []string{"PATH=/usr/bin", "LANG=C.UTF-8", "SYSTEMD_COLORS=0", "SYSTEMD_PAGER="}
	output := &boundedBuffer{Limit: 1 << 20}
	cmd.Stdout = output
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, errors.New("selected journal unavailable; check permissions or reset cursor")
	}
	return parseJournal(output.Bytes())
}

func parseJournal(raw []byte) ([]journalRecord, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	records := []journalRecord{}
	for {
		var data map[string]json.RawMessage
		if err := decoder.Decode(&data); err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.New("invalid journal response")
		}
		if len(records) >= 64 {
			return nil, errors.New("journal batch limit exceeded")
		}
		text := func(key string) string { var value string; _ = json.Unmarshal(data[key], &value); return value }
		cursor := text("__CURSOR")
		stamp, err := strconv.ParseInt(text("__REALTIME_TIMESTAMP"), 10, 64)
		if err != nil || stamp < 0 || cursor == "" || len(cursor) > 2048 {
			return nil, errors.New("invalid journal metadata")
		}
		priority, err := strconv.Atoi(text("PRIORITY"))
		if err != nil || priority < 0 || priority > 7 {
			return nil, errors.New("invalid journal priority")
		}
		records = append(records, journalRecord{cursor, text("_SYSTEMD_USER_UNIT"), stamp, priority})
	}
	return records, nil
}

func (e *Engine) sampleJournal(ctx context.Context, c Config, m Monitor, now time.Time) error {
	var state journalState
	err := e.db.QueryRowContext(ctx, "SELECT hash,cursor,since_us,next_at,last_count,error FROM journal_state WHERE id=?", m.ID).Scan(&state.Hash, &state.Cursor, &state.Since, &state.Next, &state.Count, &state.Error)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	fresh := err == sql.ErrNoRows || state.Hash != monitorHash(c, m)
	var records []journalRecord
	previousHash := state.Hash
	if fresh {
		state = journalState{Hash: monitorHash(c, m), Since: now.UnixMicro()}
	} else {
		if now.Unix() < state.Next {
			return nil
		}
		records, err = readJournal(ctx, m, state)
		state.Error = ""
		if err != nil {
			state.Error = "selected journal unavailable; check permissions or reset cursor"
		}
	}
	state.PreviousHash = previousHash
	state.Next = now.Unix() + int64(m.Interval)
	return e.commitJournal(ctx, m, state, records)
}

func (e *Engine) commitJournal(ctx context.Context, m Monitor, state journalState, records []journalRecord) error {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	c, err := e.config(ctx, "active")
	if err != nil {
		return err
	}
	valid := false
	for _, active := range c.Monitors {
		if active.ID == m.ID && active.Enabled && monitorHash(c, active) == state.Hash {
			valid = true
		}
	}
	if !valid {
		return nil
	}
	var grant string
	err = e.db.QueryRowContext(ctx, "SELECT hash FROM grants WHERE scope=?", "monitor:"+m.ID).Scan(&grant)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if grant != state.Hash {
		return nil
	}
	var persistedHash string
	err = e.db.QueryRowContext(ctx, "SELECT hash FROM journal_state WHERE id=?", m.ID).Scan(&persistedHash)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if persistedHash != state.PreviousHash {
		return nil
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	state.Count = 0
	for _, record := range records {
		// Lifecycle messages can match journalctl --unit without belonging to the
		// selected process. Advance their cursor but emit only exact unit metadata.
		if record.Unit == m.Unit && record.Priority <= m.Priority {
			_, err = e.enqueue(ctx, tx, c, Event{ID: bodyIdentity("journal", []byte(record.Cursor)), Source: "monitor:" + m.ID, Type: "journal", Data: map[string]any{"unit": m.Unit, "priority": record.Priority, "timestamp": time.UnixMicro(record.Timestamp).UTC().Format(time.RFC3339Nano)}})
			if err != nil {
				return err
			}
			state.Count++
		}
		state.Cursor = record.Cursor
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO journal_state(id,hash,cursor,since_us,next_at,last_count,error) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET hash=excluded.hash,cursor=excluded.cursor,since_us=excluded.since_us,next_at=excluded.next_at,last_count=excluded.last_count,error=excluded.error`, m.ID, state.Hash, state.Cursor, state.Since, state.Next, state.Count, state.Error)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (e *Engine) journalStatus(ctx context.Context) ([]map[string]any, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT j.id,j.last_count,j.next_at,CASE WHEN g.hash IS NULL OR g.hash!=j.hash THEN 'monitor permission revoked' ELSE j.error END FROM journal_state j LEFT JOIN grants g ON g.scope='monitor:'||j.id ORDER BY j.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, failure string
		var count int
		var next int64
		if err := rows.Scan(&id, &count, &next, &failure); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "phase": "watching", "value": count, "next": next, "error": failure, "unit": "events"})
	}
	return out, rows.Err()
}

func (e *Engine) resetMonitor(ctx context.Context, data []byte) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	if !identifier.MatchString(req.ID) {
		return nil, errors.New("invalid monitor ID")
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	c, err := e.config(ctx, "active")
	if err != nil {
		return nil, err
	}
	valid := false
	for _, m := range c.Monitors {
		if m.ID == req.ID && m.Metric == "journal" {
			valid = true
		}
	}
	if !valid {
		return nil, errors.New("reset requires an active journal monitor")
	}
	// No permissions are changed; the next granted sample starts from that time.
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO journal_state(id,hash,cursor,since_us,next_at,last_count,error) VALUES(?,?,'',0,0,0,'') ON CONFLICT(id) DO UPDATE SET hash=excluded.hash,cursor='',since_us=0,next_at=0,last_count=0,error=''`, req.ID, "reset:"+newID())
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit(at,operation,resource,result) VALUES(?,?,?,?)", time.Now().UTC().Format(time.RFC3339Nano), "journal.reset", req.ID, "restart from next sample"); err != nil {
		return nil, err
	}
	return map[string]string{"id": req.ID, "state": "restart from next sample"}, tx.Commit()
}
