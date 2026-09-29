package core

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func hostIsolation(t *testing.T) {
	t.Helper()
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("requires explicit host integration mode")
	}
}

func unitProperty(unit, key string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	raw, err := exec.CommandContext(ctx, "systemctl", "--user", "show", "--value", "--property="+key, unit+".service").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func waitHost(t *testing.T, description string, timeout time.Duration, predicate func() bool) {
	t.Helper()
	until := time.Now().Add(timeout)
	for time.Now().Before(until) {
		if predicate() {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatal("timeout: " + description)
}

func startIsolated(t *testing.T, script string, timeout int) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	unit := "quatrro-isolation-test-" + newID()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runIsolatedCommand(ctx, Action{Kind: "command", Executable: "/usr/bin/python3", Args: []string{"-c", script}, Timeout: timeout}, unit)
	}()
	t.Cleanup(func() {
		cancel()
		runBounded(context.Background(), 3*time.Second, "systemctl", "--user", "stop", unit+".service")
	})
	return unit, cancel, done
}

func readCgroup(t *testing.T, path, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(path, name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(raw))
}

func cpuCounter(t *testing.T, path, key string) int64 {
	t.Helper()
	for _, line := range strings.Split(readCgroup(t, path, "cpu.stat"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == key {
			n, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			return n
		}
	}
	t.Fatal("missing cgroup counter", key)
	return 0
}

func TestHostIsolationEffectiveCgroupAndCPU(t *testing.T) {
	hostIsolation(t)
	unit, cancel, done := startIsolated(t, "while True: pass", 12)
	var group string
	waitHost(t, "running command cgroup", 3*time.Second, func() bool {
		group = unitProperty(unit, "ControlGroup")
		return group != "" && unitProperty(unit, "ActiveState") == "active"
	})
	path := filepath.Join("/sys/fs/cgroup", group)
	for file, want := range map[string]string{"memory.max": "268435456", "memory.swap.max": "0", "pids.max": "32"} {
		if got := readCgroup(t, path, file); got != want {
			t.Fatalf("%s=%s, expected %s", file, got, want)
		}
	}
	quota := strings.Fields(readCgroup(t, path, "cpu.max"))
	if len(quota) != 2 {
		t.Fatal(quota)
	}
	q, err := strconv.ParseFloat(quota[0], 64)
	if err != nil {
		t.Fatal("CPU quota unlimited", err)
	}
	period, err := strconv.ParseFloat(quota[1], 64)
	if err != nil || q/period != 0.5 {
		t.Fatal("CPU quota differs", quota)
	}
	throttled := cpuCounter(t, path, "nr_throttled")
	usage := cpuCounter(t, path, "usage_usec")
	start := time.Now()
	time.Sleep(1500 * time.Millisecond)
	used := cpuCounter(t, path, "usage_usec") - usage
	elapsed := time.Since(start).Microseconds()
	if cpuCounter(t, path, "nr_throttled") <= throttled || float64(used) > float64(elapsed)*0.8 {
		t.Fatalf("CPU limit not enforced: usage=%dus elapsed=%dus", used, elapsed)
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancelled busy command succeeded")
	}
	waitHost(t, "cgroup removed after cancel", 3*time.Second, func() bool { _, err := os.Stat(path); return os.IsNotExist(err) })
}

func TestHostIsolationMemoryAndTaskLimits(t *testing.T) {
	hostIsolation(t)
	// The allocation is deliberately bounded to 300 MiB and protected by the
	// same 256 MiB cgroup whose kernel controls are checked in the CPU test.
	for name, script := range map[string]string{
		"memory": "import time; time.sleep(0.3); blocks=[]\nfor i in range(300): blocks.append(bytearray(1024*1024))\nraise SystemExit(0)",
		"tasks":  "import os,time\nchildren=[]\ntry:\n for i in range(40):\n  pid=os.fork()\n  if pid==0: time.sleep(10); os._exit(0)\n  children.append(pid)\nexcept OSError:\n for pid in children: os.kill(pid,9)\n raise SystemExit(0)\nfor pid in children: os.kill(pid,9)\nraise SystemExit(42)",
	} {
		t.Run(name, func(t *testing.T) {
			unit, _, done := startIsolated(t, script, 8)
			select {
			case err := <-done:
				if name == "memory" {
					if err == nil {
						t.Fatal("memory allocation exceeded cgroup limit")
					}
					waitHost(t, "kernel OOM result", 2*time.Second, func() bool {
						raw, err := exec.Command("journalctl", "--user", "--unit="+unit+".service", "--no-pager", "--output=cat").Output()
						return err == nil && strings.Contains(string(raw), "oom-kill")
					})
				}
				if name == "tasks" && err != nil {
					t.Fatal("process count was not limited", err)
				}
			case <-time.After(12 * time.Second):
				t.Fatal("resource probe hung")
			}
		})
	}
}

func TestHostIsolationFilesystemNetworkAndEnvironment(t *testing.T) {
	hostIsolation(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	marker := filepath.Join(t.TempDir(), "private-marker")
	if err = os.WriteFile(marker, []byte("fixture-private-data"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QUATRRO_PRIVATE_TEST_VALUE", "must-not-reach-command")
	script := `import os,socket,sys
assert 'QUATRRO_PRIVATE_TEST_VALUE' not in os.environ
assert 'DBUS_SESSION_BUS_ADDRESS' not in os.environ
assert 'HOME' not in os.environ
assert not os.path.exists(sys.argv[1])
assert not os.path.exists('/etc/passwd')
assert os.statvfs('/usr').f_flag & os.ST_RDONLY
assert 'NoNewPrivs:\t1' in open('/proc/self/status').read()
assert len(open('/proc/net/route').read().splitlines()) == 1
assert not os.path.exists('/run/user')
assert set(os.listdir('/sys/class/net'))=={'lo'} if os.path.exists('/sys/class/net') else True
try:
 open('/usr/quatrro-isolation-write-test','w').close()
except OSError: pass
else: raise AssertionError('writable /usr')
with open('/tmp/local-write','w') as f: f.write('private temporary file')
s=socket.socket();s.settimeout(0.5)
try: s.connect(('127.0.0.1',int(sys.argv[2])))
except OSError: pass
else: raise AssertionError('host network reachable')
`
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	a := Action{Kind: "command", Executable: "/usr/bin/python3", Args: []string{"-c", script, marker, port}, Timeout: 5}
	if err := testEngine(t).perform(context.Background(), a, Event{}); err != nil {
		t.Fatal("filesystem/network/environment probe failed", err)
	}
}

func TestHostIsolationTimeoutAndDescendantCancellation(t *testing.T) {
	hostIsolation(t)
	script := `import os,signal,time
signal.signal(signal.SIGTERM,signal.SIG_IGN)
if os.fork()==0:
 os.setsid()
 time.sleep(60)
else: time.sleep(60)
`
	for _, mode := range []string{"timeout", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			timeout := 10
			if mode == "timeout" {
				timeout = 2
			}
			unit, cancel, done := startIsolated(t, script, timeout)
			var path string
			var pids []string
			waitHost(t, "descendant started", time.Second, func() bool {
				group := unitProperty(unit, "ControlGroup")
				if group == "" {
					return false
				}
				path = filepath.Join("/sys/fs/cgroup", group)
				raw, err := os.ReadFile(filepath.Join(path, "cgroup.procs"))
				if err != nil {
					return false
				}
				pids = strings.Fields(string(raw))
				return len(pids) >= 3
			})
			start := time.Now()
			if mode == "cancel" {
				cancel()
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("terminated command succeeded")
				}
			case <-time.After(7 * time.Second):
				t.Fatal("termination exceeded bound")
			}
			if time.Since(start) > 6*time.Second {
				t.Fatal("termination too slow")
			}
			waitHost(t, "descendants removed", 2*time.Second, func() bool {
				for _, pid := range pids {
					if _, err := os.Stat("/proc/" + pid); !os.IsNotExist(err) {
						return false
					}
				}
				return true
			})
		})
	}
}
