package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type monitorState struct {
	Hash, Phase       string
	Since, Last, Next int64
	Value             float64
	Error             string
}
type cpuSample struct{ Total, Idle uint64 }

func readCPU() (cpuSample, error) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuSample{}, err
	}
	line := strings.SplitN(string(b), "\n", 2)[0]
	fields := strings.Fields(line)
	var s cpuSample
	for i, v := range fields[1:] {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return s, err
		}
		if i < 8 {
			s.Total += n
		}
		if i == 3 || i == 4 {
			s.Idle += n
		}
	}
	return s, nil
}
func (e *Engine) metric(ctx context.Context, m Monitor) (float64, error) {
	switch m.Metric {
	case "process", "file_exists", "file_age", "file_size", "temperature":
		return extendedMetric(ctx, m, time.Now())
	case "cpu":
		current, err := readCPU()
		if err != nil {
			return 0, err
		}
		previous := e.cpu[m.ID]
		e.cpu[m.ID] = current
		if previous.Total == 0 || current.Total <= previous.Total {
			return 0, errors.New("waiting for CPU sample")
		}
		return 100 * (1 - float64(current.Idle-previous.Idle)/float64(current.Total-previous.Total)), nil
	case "memory":
		b, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0, err
		}
		values := map[string]float64{}
		for _, line := range strings.Split(string(b), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				v, _ := strconv.ParseFloat(parts[1], 64)
				values[strings.TrimSuffix(parts[0], ":")] = v
			}
		}
		if values["MemTotal"] <= 0 {
			return 0, errors.New("memory unavailable")
		}
		return 100 * (1 - values["MemAvailable"]/values["MemTotal"]), nil
	case "disk":
		var s syscall.Statfs_t
		path := m.Path
		if path == "" {
			path = "/"
		}
		if err := syscall.Statfs(path, &s); err != nil {
			return 0, err
		}
		if s.Blocks == 0 {
			return 0, errors.New("disk size unavailable")
		}
		return float64(s.Bavail) / float64(s.Blocks) * 100, nil
	case "battery":
		matches, _ := filepath.Glob("/sys/class/power_supply/*/capacity")
		for _, path := range matches {
			b, err := os.ReadFile(path)
			if err == nil {
				return strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
			}
		}
		return 0, errors.New("battery unavailable")
	case "service":
		if err := runBounded(ctx, 3*time.Second, "systemctl", "--user", "is-active", m.Unit); err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) && exit.ExitCode() == 3 {
				return 0, nil
			}
			return 0, errors.New("service metric unavailable")
		}
		return 100, nil
	}
	return 0, errors.New("metric unavailable")
}
func transition(m Monitor, s monitorState, value float64, now int64) (monitorState, string) {
	low := lowMetric(m.Metric)
	alarm := value > m.Threshold
	recovered := value < m.Recovery
	if low {
		alarm = value < m.Threshold
		recovered = value > m.Recovery
	}
	s.Value = value
	s.Error = ""
	s.Next = now + int64(m.Interval)
	if s.Phase == "alert" {
		if recovered {
			if s.Since == 0 {
				s.Since = now
			}
			if now-s.Since < int64(m.RecoveryDuration) {
				return s, ""
			}
			s.Phase = "normal"
			s.Since = 0
			return s, "recovered"
		}
		s.Since = 0
		return s, ""
	}
	if !alarm {
		s.Phase = "normal"
		s.Since = 0
		return s, ""
	}
	if s.Since == 0 {
		s.Since = now
	}
	s.Phase = "pending"
	if now-s.Since >= int64(m.Duration) && (s.Last == 0 || now-s.Last >= int64(m.Cooldown)) {
		s.Phase = "alert"
		s.Since = 0
		s.Last = now
		return s, "alert"
	}
	return s, ""
}
func (e *Engine) RunMonitors(ctx context.Context) error {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			if err := e.sampleMonitors(ctx, time.Now()); err != nil && ctx.Err() == nil {
				e.lastError.Store("monitor storage failure")
			}
		}
	}
}
func (e *Engine) sampleMonitors(ctx context.Context, now time.Time) error {
	c, err := e.config(ctx, "active")
	if err != nil {
		return err
	}
	for _, m := range c.Monitors {
		if !m.Enabled {
			continue
		}
		var grant string
		if err = e.db.QueryRowContext(ctx, "SELECT hash FROM grants WHERE scope=?", "monitor:"+m.ID).Scan(&grant); err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return err
		}
		if grant != monitorHash(c, m) {
			continue
		}
		if m.Metric == "journal" {
			if err = e.sampleJournal(ctx, c, m, now); err != nil {
				return err
			}
			continue
		}
		var s monitorState
		err := e.db.QueryRowContext(ctx, "SELECT hash,phase,since,last,next,value,error FROM monitor_state WHERE id=?", m.ID).Scan(&s.Hash, &s.Phase, &s.Since, &s.Last, &s.Next, &s.Value, &s.Error)
		if err == sql.ErrNoRows {
			s = monitorState{}
		} else if err != nil {
			return err
		}
		hash := monitorHash(c, m)
		if s.Hash != hash {
			s = monitorState{Hash: hash, Phase: "normal"}
		}
		if now.Unix() < s.Next {
			continue
		}
		if s.Next > 0 && now.Unix()-s.Next > int64(m.Interval*3) {
			s.Since = 0
		}
		var value float64
		var readErr error
		if m.Metric == "connectivity" {
			value, readErr = e.connectivityMetric(ctx, c, m, nil)
		} else {
			value, readErr = e.metric(ctx, m)
		}
		kind := ""
		if readErr != nil {
			s.Error = "metric unavailable"
			s.Since = 0
			s.Next = now.Unix() + int64(m.Interval)
		} else {
			s, kind = transition(m, s, value, now.Unix())
		}
		if err = e.commitMonitor(ctx, m, s, kind, now); err != nil {
			return err
		}
	}
	return nil
}
func (e *Engine) monitorStatus(ctx context.Context) (any, error) {
	c, err := e.config(ctx, "active")
	if err != nil {
		return nil, err
	}
	units := map[string]string{}
	for _, m := range c.Monitors {
		units[m.ID] = metricUnit(m.Metric)
	}
	rows, err := e.db.QueryContext(ctx, "SELECT m.id,m.phase,m.value,CASE WHEN g.hash IS NULL OR g.hash!=m.hash THEN 'monitor permission revoked' ELSE m.error END,m.next FROM monitor_state m LEFT JOIN grants g ON g.scope='monitor:'||m.id ORDER BY m.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, phase, failure string
		var value float64
		var next int64
		if err := rows.Scan(&id, &phase, &value, &failure, &next); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "phase": phase, "value": value, "error": failure, "next": next, "unit": units[id]})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	journal, err := e.journalStatus(ctx)
	if err != nil {
		return nil, err
	}
	return append(out, journal...), nil
}

func (e *Engine) commitMonitor(ctx context.Context, m Monitor, s monitorState, kind string, now time.Time) error {
	e.mutation.Lock()
	defer e.mutation.Unlock()
	c, err := e.config(ctx, "active")
	if err != nil {
		return err
	}
	valid := false
	for _, active := range c.Monitors {
		if active.ID == m.ID && active.Enabled && monitorHash(c, active) == s.Hash && canonicalHash(active) == canonicalHash(m) {
			valid = true
			break
		}
	}
	if !valid {
		return nil
	}
	var grant string
	if err = e.db.QueryRowContext(ctx, "SELECT hash FROM grants WHERE scope=?", "monitor:"+m.ID).Scan(&grant); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	if grant != s.Hash {
		return nil
	}
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if kind != "" {
		_, err = e.enqueue(ctx, tx, c, Event{ID: fmt.Sprintf("%s:%s:%d", m.ID, kind, now.Unix()), Source: "monitor:" + m.ID, Type: kind, Data: map[string]any{"metric": m.Metric, "value": s.Value, "state": kind}})
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO monitor_state(id,hash,phase,since,last,next,value,error) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET hash=excluded.hash,phase=excluded.phase,since=excluded.since,last=excluded.last,next=excluded.next,value=excluded.value,error=excluded.error", m.ID, s.Hash, s.Phase, s.Since, s.Last, s.Next, s.Value, s.Error)
	if err != nil {
		return err
	}
	return tx.Commit()
}
