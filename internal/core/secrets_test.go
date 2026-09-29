package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCredentialRotationDeletionAndBackendIsolation(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	put := func(value, backend string) error {
		raw, _ := json.Marshal(map[string]string{"id": "test", "value": value, "backend": backend})
		_, err := e.putSecret(ctx, raw)
		return err
	}
	if err := put("first-public-test-secret", "file"); err != nil {
		t.Fatal(err)
	}
	if err := put("second-public-test-secret", "file"); err != nil {
		t.Fatal(err)
	}
	value, err := e.secret(ctx, "test")
	if err != nil || value != "second-public-test-secret" {
		t.Fatal("rotation failed", err)
	}
	if err = put("third-public-test-secret", "keyring"); err == nil {
		t.Fatal("backend switch would leave stale copy")
	}
	if _, err = e.deleteSecret(ctx, []byte(`{"id":"test"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = e.secret(ctx, "test"); err == nil {
		t.Fatal("deleted secret still accessible")
	}
	if _, err = os.Stat(filepath.Join(e.paths.Config, "secrets", "test")); !os.IsNotExist(err) {
		t.Fatal("credential file retained")
	}
}

func TestCredentialFileBoundaries(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	fixture := "private-boundary-test-only-value"
	raw, _ := json.Marshal(secretWrite{ID: "boundary", Backend: "file", Value: fixture})
	if _, err := e.putSecret(ctx, raw); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(e.paths.Config, "secrets")
	path := filepath.Join(dir, "boundary")
	for name, mode := range map[string]os.FileMode{dir: 0700, path: 0600} {
		info, err := os.Stat(name)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatal("credential permissions", name, err)
		}
	}
	for _, kind := range []string{"permissions", "symlink", "directory", "oversize", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			var err error
			switch kind {
			case "permissions":
				err = os.WriteFile(path, []byte(fixture), 0600)
				if err == nil {
					err = os.Chmod(path, 0644)
				}
			case "symlink":
				target := filepath.Join(t.TempDir(), "target")
				err = os.WriteFile(target, []byte(fixture), 0600)
				if err == nil {
					err = os.Symlink(target, path)
				}
			case "directory":
				err = os.Mkdir(path, 0700)
			case "oversize":
				err = os.WriteFile(path, []byte(strings.Repeat("x", 8193)), 0600)
			case "fifo":
				err = syscall.Mkfifo(path, 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				value, err := e.secret(ctx, "boundary")
				if value != "" {
					done <- nil
					return
				}
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil || strings.Contains(err.Error(), fixture) {
					t.Fatal("unsafe credential accepted or disclosed", kind)
				}
			case <-time.After(time.Second):
				// Unblock a regressed FIFO reader so the test does not leak it.
				if kind == "fifo" {
					f, _ := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0)
					if f != nil {
						f.Close()
					}
				}
				t.Fatal("credential read blocked", kind)
			}
		})
	}
}
