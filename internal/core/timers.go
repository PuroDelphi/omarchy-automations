package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

type Timer struct {
	ID       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Kind     string   `json:"kind"`
	Interval int      `json:"interval_seconds,omitempty"`
	At       string   `json:"at,omitempty"`
	Timezone string   `json:"timezone,omitempty"`
	Weekdays []string `json:"weekdays,omitempty"`
	Missed   string   `json:"missed"`
	Enabled  bool     `json:"enabled"`
}

var weekdayNames = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

func (t Timer) Validate() error {
	if t.Missed != "coalesce" && t.Missed != "skip" {
		return errors.New("choose missed policy: coalesce or skip")
	}
	switch t.Kind {
	case "interval":
		if t.Interval < 5 || t.Interval > 31536000 {
			return errors.New("interval must be 5..31536000 seconds")
		}
		if t.At != "" || t.Timezone != "" || len(t.Weekdays) > 0 {
			return errors.New("calendar fields are not allowed for intervals")
		}
	case "calendar":
		if t.Interval != 0 {
			return errors.New("calendar cannot have interval_seconds")
		}
		at, err := time.Parse("15:04", t.At)
		if err != nil || at.Format("15:04") != t.At {
			return errors.New("calendar time must be HH:MM")
		}
		if t.Timezone == "" || t.Timezone == "Local" {
			return errors.New("choose an explicit IANA timezone or UTC")
		}
		if _, err := time.LoadLocation(t.Timezone); err != nil {
			return errors.New("unknown timezone")
		}
		seen := map[string]bool{}
		for _, day := range t.Weekdays {
			found := false
			for _, name := range weekdayNames {
				if day == name {
					found = true
				}
			}
			if !found || seen[day] {
				return errors.New("weekdays must be unique sun/mon/tue/wed/thu/fri/sat")
			}
			seen[day] = true
		}
	default:
		return errors.New("unknown timer kind")
	}
	return nil
}

func calendarKey(t Timer, instant time.Time) string {
	zone, _ := time.LoadLocation(t.Timezone)
	return instant.In(zone).Format("2006-01-02T15:04")
}

// Scan UTC minutes so nonexistent local times are skipped. Excluding the last
// local occurrence prevents running twice during a repeated DST hour.
func nextTimer(t Timer, after time.Time, lastKey string) (time.Time, error) {
	if t.Kind == "interval" {
		return after.Add(time.Duration(t.Interval) * time.Second), nil
	}
	zone, err := time.LoadLocation(t.Timezone)
	if err != nil {
		return time.Time{}, err
	}
	for candidate, end := after.UTC().Truncate(time.Minute).Add(time.Minute), after.Add(8*24*time.Hour); candidate.Before(end); candidate = candidate.Add(time.Minute) {
		local := candidate.In(zone)
		if local.Format("15:04") != t.At || local.Format("2006-01-02T15:04") == lastKey {
			continue
		}
		// Only the first occurrence of a repeated wall-clock minute is eligible,
		// even when the engine starts inside the second occurrence's hour.
		transition, _ := local.ZoneBounds()
		if !transition.IsZero() {
			_, offset := local.Zone()
			_, previous := transition.Add(-time.Nanosecond).In(zone).Zone()
			if previous > offset && candidate.Before(transition.Add(time.Duration(previous-offset)*time.Second)) {
				continue
			}
		}
		if len(t.Weekdays) > 0 {
			found := false
			for _, day := range t.Weekdays {
				if weekdayNames[int(local.Weekday())] == day {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		return candidate, nil
	}
	return time.Time{}, errors.New("no calendar occurrence in next eight days")
}

type timerState struct {
	Hash    string
	Next    int64
	LastKey string
	Last    int64
}

func (e *Engine) RunTimers(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := e.scheduleTimers(ctx, time.Now()); err != nil && ctx.Err() == nil {
				e.lastError.Store("scheduler storage failure")
			}
		}
	}
}

func (e *Engine) scheduleTimers(ctx context.Context, now time.Time) error {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	c, err := e.config(ctx, "active")
	if err != nil {
		return err
	}
	for _, t := range c.Timers {
		if !t.Enabled {
			continue
		}
		hash := canonicalHash(t)
		var grant string
		err = e.db.QueryRowContext(ctx, "SELECT hash FROM grants WHERE scope=?", "timer:"+t.ID).Scan(&grant)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		if grant != hash {
			continue
		}
		var state timerState
		err = e.db.QueryRowContext(ctx, "SELECT hash,next_at,last_key,last_at FROM timer_state WHERE id=?", t.ID).Scan(&state.Hash, &state.Next, &state.LastKey, &state.Last)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		fresh := err == sql.ErrNoRows || state.Hash != hash
		if !fresh && state.Next > now.Unix() {
			continue
		}
		tx, err := e.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		err = e.advanceTimer(ctx, tx, c, t, state, fresh, now)
		if err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) advanceTimer(ctx context.Context, tx *sql.Tx, c Config, t Timer, state timerState, fresh bool, now time.Time) error {
	hash := canonicalHash(t)
	if fresh {
		state = timerState{Hash: hash}
	} else {
		due := time.Unix(state.Next, 0)
		grace := int64(5)
		if t.Kind == "calendar" {
			grace = 60
		}
		if t.Missed == "coalesce" || now.Unix()-state.Next <= grace {
			_, err := e.enqueue(ctx, tx, c, Event{ID: fmt.Sprintf("%s:%s:%d", t.ID, hash, state.Next), Source: "timer:" + t.ID, Type: "scheduled", Data: map[string]any{"timer": t.ID, "scheduled_at": due.UTC().Format(time.RFC3339), "fired_at": now.UTC().Format(time.RFC3339), "late_seconds": now.Unix() - state.Next, "policy": t.Missed}})
			if err != nil {
				return err
			}
			state.Last = now.Unix()
		}
		// A skipped occurrence is consumed too, including its repeated DST hour.
		if t.Kind == "calendar" {
			state.LastKey = calendarKey(t, due)
		}
	}
	next, err := nextTimer(t, now, state.LastKey)
	if err != nil {
		return err
	}
	if !fresh && t.Kind == "interval" {
		// Preserve phase but consolidate all overdue intervals into at most one event.
		next = time.Unix(state.Next+((now.Unix()-state.Next)/int64(t.Interval)+1)*int64(t.Interval), 0)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO timer_state(id,hash,next_at,last_key,last_at) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET hash=excluded.hash,next_at=excluded.next_at,last_key=excluded.last_key,last_at=excluded.last_at`, t.ID, hash, next.Unix(), state.LastKey, state.Last)
	return err
}

func (e *Engine) timerStatus(ctx context.Context) (any, error) {
	c, err := e.config(ctx, "active")
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, t := range c.Timers {
		var state timerState
		err = e.db.QueryRowContext(ctx, "SELECT hash,next_at,last_key,last_at FROM timer_state WHERE id=?", t.ID).Scan(&state.Hash, &state.Next, &state.LastKey, &state.Last)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		phase := "scheduled"
		var grant string
		grantErr := e.db.QueryRowContext(ctx, "SELECT hash FROM grants WHERE scope=?", "timer:"+t.ID).Scan(&grant)
		if grantErr != nil && grantErr != sql.ErrNoRows {
			return nil, grantErr
		}
		if !t.Enabled {
			phase = "disabled"
		} else if grant != canonicalHash(t) {
			phase = "permission revoked"
		} else if state.Hash != canonicalHash(t) {
			phase = "initializing"
		}
		out = append(out, map[string]any{"id": t.ID, "phase": phase, "next": state.Next, "last": state.Last, "timezone": t.Timezone, "weekdays": strings.Join(t.Weekdays, ",")})
	}
	return out, nil
}
