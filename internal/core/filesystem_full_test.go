package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"modernc.org/sqlite"
)

// The filler can only run on a small private tmpfs, never on the host disk.
func TestFilesystemFullRecovery(t *testing.T) {
	if os.Getenv("QUATRRO_ENOSPC_CHILD") != "1" {
		if os.Getenv("QUATRRO_HOST_TEST") != "1" {
			t.Skip("requires opt-in host namespaces")
		}
		binary, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "bwrap", "--unshare-all", "--die-with-parent",
			"--ro-bind", "/", "/", "--ro-bind", binary, "/usr/bin/true",
			"--size", "16777216", "--tmpfs", "/tmp", "--proc", "/proc", "--dev", "/dev",
			"--setenv", "TMPDIR", "/tmp", "--setenv", "QUATRRO_ENOSPC_CHILD", "1",
			"/usr/bin/true", "-test.run=^TestFilesystemFullRecovery$", "-test.v")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("isolated test: %v\n%s", err, out)
		}
		t.Log(string(out))
		return
	}
	var fs syscall.Statfs_t
	if err := syscall.Statfs("/tmp", &fs); err != nil {
		t.Fatal(err)
	}
	if fs.Type != 0x01021994 || fs.Blocks*uint64(fs.Bsize) > 16<<20 {
		t.Fatal("refusing filler outside limited tmpfs")
	}
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	var mode string
	if err := e.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatal("expected WAL", mode, err)
	}
	if _, err := e.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		t.Fatal(err)
	}
	filler := filepath.Join("/tmp", "space-filler")
	f, err := os.Create(filler)
	if err != nil {
		t.Fatal(err)
	}
	block := make([]byte, 4096)
	for i := 0; i <= 4096; i++ {
		_, err = f.Write(block)
		if err != nil {
			break
		}
	}
	f.Close()
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatal("expected actual filesystem ENOSPC", err)
	}
	event := Event{ID: "filesystem-full", Source: "local:test", Data: map[string]any{"blob": strings.Repeat("x", 200000)}}
	result, err := e.ingest(ctx, event)
	var storageError *sqlite.Error
	if !errors.As(err, &storageError) || (storageError.Code()&255 != 13 && storageError.Code()&255 != 10) {
		t.Fatalf("expected SQLite FULL or IOERR on full filesystem: result=%#v error=%v", result, err)
	}
	for _, table := range []string{"events", "executions", "seen_deliveries"} {
		var n int
		if err := e.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatal("partial transaction", table, n, err)
		}
	}
	if err := os.Remove(filler); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ingest(ctx, event); err != nil {
		t.Fatal("retry after freeing space", err)
	}
	p := e.paths
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var integrity string
	if err := reopened.db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatal("integrity after recovery", integrity, err)
	}
	if result, err := reopened.ingest(ctx, event); err != nil || result.(map[string]any)["duplicate"] != true {
		t.Fatal("lost durable deduplication", result, err)
	}
	var n int
	if err := reopened.db.QueryRow("SELECT count(*) FROM executions").Scan(&n); err != nil || n != 1 {
		t.Fatal("retry must produce exactly one execution", n, err)
	}
}
