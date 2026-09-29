package broker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// Only invoked by scripts/test-broker-systemd.py inside its ephemeral root.
func TestSystemdBrokerFixture(t *testing.T) {
	mode := os.Getenv("QUATRRO_BROKER_FIXTURE")
	if mode == "" {
		t.Skip("isolated systemd harness only")
	}
	mapping, e := os.ReadFile("/proc/self/uid_map")
	fields := strings.Fields(string(mapping))
	if e != nil || len(fields) < 3 || fields[0] != "0" || fields[1] == "0" {
		t.Fatal("fixture requires subordinate user namespace")
	}
	if mode == "prepare" {
		if os.Getuid() != 0 {
			t.Fatal("prepare requires namespace root")
		}

		p := Policy{Version: 1, Rules: []Rule{{UID: 1000, Unit: "quatrro-fixture.service", Operations: []string{"status", "start", "stop", "restart"}}}}
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		rules, err := RenderPolkitRules(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, dir := range []string{"/etc/quatrro", "/etc/polkit-1/rules.d", "/run/polkit-1/rules.d", "/usr/local/share/polkit-1/rules.d"} {
			if err = os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err = os.WriteFile(DefaultPolicyPath, raw, 0600); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile("/etc/polkit-1/rules.d/00-quatrro.rules", rules, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if mode != "call" {
		t.Fatal("unknown fixture mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	unit := os.Getenv("QUATRRO_TEST_UNIT")
	if unit == "" {
		unit = "quatrro-fixture.service"
	}
	result, err := Call(ctx, Request{Version: 1, ID: "systemd-fixture", Unit: unit, Operation: os.Getenv("QUATRRO_TEST_OPERATION"), CheckOnly: os.Getenv("QUATRRO_TEST_CHECK") == "1"})
	if os.Getenv("QUATRRO_TEST_DENIED") == "1" {
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("want denied, got %v", err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("QUATRRO_TEST_CHECK") == "1" {
		if result.State != "authorized" {
			t.Fatal("check did not return authorization")
		}
		return
	}
	if state := os.Getenv("QUATRRO_TEST_STATE"); state != "" && result.ActiveState != state {
		t.Fatalf("state %q want %q", result.ActiveState, state)
	}
}
