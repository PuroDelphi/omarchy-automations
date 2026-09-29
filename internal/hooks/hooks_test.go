package hooks

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestContracts(t *testing.T) {
	for name, args := range map[string][]string{"theme-set": {"a\";$(touch never)"}, "font-set": {"JetBrains Mono"}, "battery-low": {"17"}, "post-boot": nil, "post-update": nil, "pre-refresh-pacman": nil} {
		raw, err := Payload(name, args)
		if err != nil {
			t.Fatal(err)
		}
		var event struct {
			ID, Source, Type string
			Data             map[string]any
		}
		if err = json.Unmarshal(raw, &event); err != nil {
			t.Fatal(err)
		}
		if event.Source != "hook:"+name || event.Type != "omarchy."+name || len(event.ID) != 32 {
			t.Fatalf("bad envelope: %s", raw)
		}
		if name == "theme-set" && event.Data["theme"] != args[0] {
			t.Fatal("argument changed")
		}
		if name == "battery-low" && event.Data["percentage"] != float64(17) {
			t.Fatal("battery must be numeric")
		}
		next, _ := Payload(name, args)
		if string(next) == string(raw) {
			t.Fatal("reused invocation identity")
		}
	}
	for _, tc := range []struct {
		name string
		args []string
	}{{"unknown", nil}, {"theme-set", nil}, {"post-boot", []string{"extra"}}, {"battery-low", []string{"101"}}, {"battery-low", []string{"-1"}}, {"battery-low", []string{"1;command"}}} {
		if _, err := Payload(tc.name, tc.args); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
}

func TestUnavailableAndStalledBackend(t *testing.T) {
	socket := filepath.Join(t.TempDir(), "socket")
	start := time.Now()
	if err := Send(context.Background(), socket, "post-boot", nil); err == nil {
		t.Fatal("missing backend accepted")
	}
	if time.Since(start) > Timeout {
		t.Fatal("missing backend blocked")
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		for {
			if _, err = conn.Read(buf); err != nil {
				return
			}
		}
	}()
	start = time.Now()
	if err = Send(context.Background(), socket, "post-boot", nil); err == nil {
		t.Fatal("stalled backend accepted")
	}
	if elapsed := time.Since(start); elapsed > 2*Timeout {
		t.Fatalf("hook blocked for %s", elapsed)
	}
	<-done
}
