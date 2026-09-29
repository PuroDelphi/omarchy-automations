package core

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"quatrro.local/automations/internal/paths"
)

type secretWrite struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Backend string `json:"backend"`
}

// Values never appear in argv, responses, configuration exports, or audit records.
func (e *Engine) putSecret(ctx context.Context, data []byte) (any, error) {
	var req secretWrite
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	if !identifier.MatchString(req.ID) || len(req.Value) < 16 || len(req.Value) > 8192 || strings.ContainsAny(req.Value, "\r\n\x00") {
		return nil, errors.New("secret requires a valid ID and 16..8192 bytes without CR, LF or NUL")
	}
	if req.Backend != "file" && req.Backend != "keyring" {
		return nil, errors.New("select keyring or explicit plaintext file storage")
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	return e.putSecretLocked(ctx, req)
}

// Caller holds mutation; used for atomic credential compare-and-replace.
func (e *Engine) putSecretLocked(ctx context.Context, req secretWrite) (any, error) {
	var previous string
	lookupErr := e.db.QueryRowContext(ctx, "SELECT backend FROM secrets WHERE id=?", req.ID).Scan(&previous)
	if lookupErr != nil && lookupErr != sql.ErrNoRows {
		return nil, lookupErr
	}
	if previous != "" && previous != req.Backend {
		return nil, errors.New("credential uses another backend; delete it before changing storage")
	}
	if req.Backend == "keyring" {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "secret-tool", "store", "--label=Omarchy Automations "+req.ID, "application", "quatrro", "profile", e.paths.Config, "id", req.ID)
		cmd.Stdin = strings.NewReader(req.Value)
		if err := cmd.Run(); err != nil {
			return nil, errors.New("keyring unavailable or locked; secret was not saved")
		}
	} else {
		dir := filepath.Join(e.paths.Config, "secrets")
		if err := paths.PrivateDir(dir); err != nil {
			return nil, err
		}
		f, err := os.CreateTemp(dir, ".pending-")
		if err != nil {
			return nil, err
		}
		name := f.Name()
		defer os.Remove(name)
		if _, err = f.WriteString(req.Value); err != nil {
			f.Close()
			return nil, err
		}
		if err = f.Sync(); err != nil {
			f.Close()
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		if err = os.Rename(name, filepath.Join(dir, req.ID)); err != nil {
			return nil, err
		}
	}
	_, err := e.db.ExecContext(ctx, "INSERT INTO secrets(id,backend) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET backend=excluded.backend", req.ID, req.Backend)
	return map[string]string{"id": req.ID, "backend": req.Backend}, err
}

func (e *Engine) deleteSecret(ctx context.Context, data []byte) (any, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decode(data, &req); err != nil {
		return nil, err
	}
	if !identifier.MatchString(req.ID) {
		return nil, errors.New("invalid secret ID")
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	var backend string
	err := e.db.QueryRowContext(ctx, "SELECT backend FROM secrets WHERE id=?", req.ID).Scan(&backend)
	if err == sql.ErrNoRows {
		return map[string]string{"deleted": req.ID}, nil
	}
	if err != nil {
		return nil, err
	}
	if backend == "keyring" {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err = exec.CommandContext(ctx, "secret-tool", "clear", "application", "quatrro", "profile", e.paths.Config, "id", req.ID).Run(); err != nil {
			return nil, errors.New("keyring unavailable; credential not removed")
		}
	} else {
		dir := filepath.Join(e.paths.Config, "secrets")
		if err = paths.PrivateDir(dir); err != nil {
			return nil, err
		}
		if err = os.Remove(filepath.Join(dir, req.ID)); err != nil && !os.IsNotExist(err) {
			return nil, errors.New("credential file could not be removed")
		}
	}
	_, err = e.db.ExecContext(ctx, "DELETE FROM secrets WHERE id=?", req.ID)
	return map[string]string{"deleted": req.ID}, err
}
func (e *Engine) secret(ctx context.Context, id string) (string, error) {
	if !identifier.MatchString(id) {
		return "", errors.New("invalid secret reference")
	}
	var backend string
	if err := e.db.QueryRowContext(ctx, "SELECT backend FROM secrets WHERE id=?", id).Scan(&backend); err != nil {
		return "", errors.New("secret unavailable")
	}
	if backend == "keyring" {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		b, err := exec.CommandContext(ctx, "secret-tool", "lookup", "application", "quatrro", "profile", e.paths.Config, "id", id).Output()
		if err != nil {
			return "", errors.New("keyring unavailable or locked")
		}
		return strings.TrimSuffix(string(b), "\n"), nil
	}
	dir := filepath.Join(e.paths.Config, "secrets")
	if err := paths.PrivateDir(dir); err != nil {
		return "", err
	}
	// Open without blocking before fstat rejects special files such as FIFOs.
	f, err := os.OpenFile(filepath.Join(dir, id), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", errors.New("secret unavailable")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Getuid()) || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return "", errors.New("unsafe secret file permissions")
	}
	b, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil || len(b) > 8192 {
		return "", errors.New("invalid secret file")
	}
	return string(b), nil
}
func (e *Engine) listSecrets(ctx context.Context) (any, error) {
	rows, err := e.db.QueryContext(ctx, "SELECT id,backend FROM secrets ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var id, backend string
		if err := rows.Scan(&id, &backend); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{"id": id, "backend": backend})
	}
	return out, rows.Err()
}
