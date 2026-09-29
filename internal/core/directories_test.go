package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"quatrro.local/automations/internal/sandbox"
)

// The production helper lives in quatrrod. Host tests spawn this test binary
// through the identical os.Executable path and dispatch only its private mode.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--sandbox-runner" {
		if err := sandbox.RunIO(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func directoryFixture(t *testing.T) sandbox.Directory {
	t.Helper()
	source := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"source": source, "target": "/work/data", "access": "rw"})
	prepared, err := prepareDirectory(raw)
	if err != nil {
		t.Fatal(err)
	}
	return prepared.(sandbox.Directory)
}

func TestDirectoryCapabilityAndActionValidation(t *testing.T) {
	c := scriptConfig("exit 0\n", "bash")
	d := directoryFixture(t)
	c.Actions[0].Directories = []sandbox.Directory{d}
	c.Actions[0].WorkingDirectory = d.Target
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	scope := actionScope(c.Flows[0].ID, c.Actions[0].ID)
	original := capabilities(c)[scope]
	c.Actions[0].Directories[0].Access = "ro"
	if capabilities(c)[scope] == original {
		t.Fatal("changed access retained capability")
	}
	c.Actions[0].WorkingDirectory = "/home"
	if c.Validate() == nil {
		t.Fatal("unapproved cwd accepted")
	}
	c.Actions[0].WorkingDirectory = d.Target
	c.Actions[0].Kind = "notify"
	if c.Validate() == nil {
		t.Fatal("notification accepted mounts")
	}
}

func TestHostDirectoryActionsAndRevocation(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("host test")
	}
	for _, scenario := range []string{"script", "command", "revoked", "changed-access", "replaced"} {
		t.Run(scenario, func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			d := directoryFixture(t)
			c := scriptConfig("printf '%s' \"$1\" > result\n", "bash")
			c.Actions[0].Directories = []sandbox.Directory{d}
			c.Actions[0].WorkingDirectory = d.Target
			if scenario == "command" {
				c.Actions[0] = Action{ID: "notice", Kind: "command", Executable: "/usr/bin/touch", Args: []string{"result"}, Timeout: 5, Directories: []sandbox.Directory{d}, WorkingDirectory: d.Target}
			}
			activateTest(t, e, c)
			if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source, Data: map[string]any{"message": "mounted"}}); err != nil {
				t.Fatal(err)
			}
			expected := "completed"
			switch scenario {
			case "revoked":
				raw, _ := json.Marshal(map[string]string{"scope": actionScope(c.Flows[0].ID, "notice")})
				if _, err := e.revoke(ctx, raw); err != nil {
					t.Fatal(err)
				}
				expected = "denied"
			case "changed-access":
				c.Actions[0].Directories[0].Access = "ro"
				activateTest(t, e, c)
				expected = "denied"
			case "replaced":
				if err := os.Rename(d.Source, d.Source+"-old"); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(d.Source, 0700); err != nil {
					t.Fatal(err)
				}
				expected = "failed"
			}
			for range 2 {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var state string
			if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil || state != expected {
				t.Fatal(state, expected, err)
			}
			data, err := os.ReadFile(filepath.Join(d.Source, "result"))
			if expected == "completed" {
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "script" && string(data) != "mounted" {
					t.Fatal(string(data))
				}
			} else if !os.IsNotExist(err) {
				t.Fatal("effect occurred", err)
			}
		})
	}
}

func TestPreflightRejectsReplacedDirectory(t *testing.T) {
	e := testEngine(t)
	// Isolate identity refusal from unrelated host capability probes.
	e.actionRunner = func(context.Context, Action, Event) error { return nil }
	d := directoryFixture(t)
	c := scriptConfig("exit 0\n", "bash")
	c.Actions[0].Directories = []sandbox.Directory{d}
	if err := os.Rename(d.Source, d.Source+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(d.Source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := e.preflight(context.Background(), c); err == nil {
		t.Fatal("activation accepted replaced directory")
	}
	var count int
	if err := e.db.QueryRow("SELECT count(*) FROM grants").Scan(&count); err != nil || count != 0 {
		t.Fatal(count, err)
	}
}

const mountedWriter = `import os,signal,time
signal.signal(signal.SIGTERM,signal.SIG_IGN)
if os.fork()==0:
 os.setsid()
 with open('writes','ab',buffering=0) as output:
  while True:
   output.write(b'x')
   time.sleep(0.02)
else:
 time.sleep(60)
`

func TestHostMountedScriptTimeoutAndDescendants(t *testing.T) {
	hostIsolation(t)
	for _, mode := range []string{"timeout", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			d := directoryFixture(t)
			unit := "quatrro-mount-stop-" + newID()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			timeout := 10
			if mode == "timeout" {
				timeout = 3
			}
			action := Action{Kind: "command", Executable: "/usr/bin/python3", Args: []string{"-I", "-S", "--", "/quatrro/script"}, Timeout: timeout, Directories: []sandbox.Directory{d}, WorkingDirectory: d.Target}
			done := make(chan error, 1)
			go func() { done <- runIsolatedInput(ctx, action, unit, mountedWriter) }()
			t.Cleanup(func() {
				cancel()
				_ = runBounded(context.Background(), 3*time.Second, "systemctl", "--user", "stop", unit+".service")
			})
			var pids []string
			waitHost(t, "mounted descendant writing", 2*time.Second, func() bool {
				info, err := os.Stat(filepath.Join(d.Source, "writes"))
				if err != nil || info.Size() == 0 {
					return false
				}
				group := unitProperty(unit, "ControlGroup")
				if group == "" {
					return false
				}
				raw, err := os.ReadFile(filepath.Join("/sys/fs/cgroup", group, "cgroup.procs"))
				if err != nil {
					return false
				}
				pids = strings.Fields(string(raw))
				return len(pids) >= 4
			})
			if mode == "cancel" {
				cancel()
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("interrupted script reported success")
				}
			case <-time.After(8 * time.Second):
				t.Fatal("termination exceeded bound")
			}
			waitHost(t, "all mounted script processes removed", 2*time.Second, func() bool {
				for _, pid := range pids {
					if _, err := os.Stat("/proc/" + pid); !os.IsNotExist(err) {
						return false
					}
				}
				return true
			})
			before, err := os.ReadFile(filepath.Join(d.Source, "writes"))
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(150 * time.Millisecond)
			after, err := os.ReadFile(filepath.Join(d.Source, "writes"))
			if err != nil || string(before) != string(after) {
				t.Fatal("writes continued after termination", err)
			}
		})
	}
}

func TestHostCancelledMountedScriptStaysUncertain(t *testing.T) {
	hostIsolation(t)
	e := testEngine(t)
	d := directoryFixture(t)
	c := scriptConfig(mountedWriter, "python3")
	c.Actions[0].Directories = []sandbox.Directory{d}
	c.Actions[0].WorkingDirectory = d.Target
	c.Actions[0].Timeout = 20
	activateTest(t, e, c)
	ctx := context.Background()
	if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source, Data: map[string]any{"message": "ignored"}}); err != nil {
		t.Fatal(err)
	}
	work, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- e.tick(work) }()
	waitHost(t, "queued script writing", 3*time.Second, func() bool {
		info, err := os.Stat(filepath.Join(d.Source, "writes"))
		return err == nil && info.Size() > 0
	})
	if _, err := e.cancelJobs(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("worker did not stop")
	}
	var state string
	if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil || state != "uncertain" {
		t.Fatal(state, err)
	}
	before, err := os.ReadFile(filepath.Join(d.Source, "writes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := e.tick(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	after, err := os.ReadFile(filepath.Join(d.Source, "writes"))
	if err != nil || string(before) != string(after) {
		t.Fatal("cancelled job repeated or kept writing", err)
	}
}
