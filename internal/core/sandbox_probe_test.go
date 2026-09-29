package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxProbeRejectsMissingOrUnlimitedControllers(t *testing.T) {
	for _, bad := range []string{"", "cpu.max", "memory.max", "memory.swap.max", "pids.max"} {
		path := t.TempDir()
		for name, value := range map[string]string{"cpu.max": "50000 100000", "memory.max": "268435456", "memory.swap.max": "0", "pids.max": "32"} {
			if name == bad {
				value = "max"
			}
			if err := os.WriteFile(filepath.Join(path, name), []byte(value), 0600); err != nil {
				t.Fatal(err)
			}
		}
		err := checkSandboxLimits(path)
		if bad == "" && err != nil {
			t.Fatal(err)
		}
		if bad != "" && err == nil {
			t.Fatal("unlimited control accepted", bad)
		}
	}
	if err := checkSandboxLimits(t.TempDir()); err == nil {
		t.Fatal("absent controllers accepted")
	}
}

func TestHostCommandActivationPreflight(t *testing.T) {
	hostIsolation(t)
	c := notificationConfig()
	c.Actions[0] = Action{ID: "notice", Kind: "command", Executable: "/usr/bin/true", Timeout: 5}
	e := testEngine(t)
	activateTest(t, e, c)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := verifyCommandSandbox(ctx); err == nil {
		t.Fatal("cancelled preflight succeeded")
	}
}

func TestCommandArgumentsRejectNULAndNoncanonicalPaths(t *testing.T) {
	for _, a := range []Action{
		{ID: "notice", Kind: "command", Executable: "/usr/bin/../bin/true", Timeout: 5},
		{ID: "notice", Kind: "command", Executable: "/usr/bin/true\x00", Timeout: 5},
		{ID: "notice", Kind: "command", Executable: "/usr/bin/echo", Args: []string{"a\x00b"}, Timeout: 5},
	} {
		c := notificationConfig()
		c.Actions[0] = a
		if err := c.Validate(); err == nil {
			t.Fatal("invalid command accepted", a.Executable)
		}
	}
}
