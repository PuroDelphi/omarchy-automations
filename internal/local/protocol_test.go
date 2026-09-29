package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestControlSocketVersionAndSingleton(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, path, func(_ context.Context, r Request) (any, error) { return map[string]string{"operation": r.Op}, nil })
	}()
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket not ready")
		}
		time.Sleep(time.Millisecond)
	}
	res, err := Call(ctx, path, Request{Op: "status"})
	if err != nil || !res.OK {
		t.Fatal(res, err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("socket permissions")
	}
	if err := Serve(ctx, path, nil); err == nil {
		t.Fatal("second daemon accepted")
	}
	if _, err := Call(ctx, path, Request{Op: "still-alive"}); err != nil {
		t.Fatal("second daemon disrupted first", err)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
