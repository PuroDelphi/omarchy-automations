package broker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Opt-in integration test: real polkitd, private bus, subordinate user IDs,
// read-only host filesystem and no host network or runtime sockets.
func TestIsolatedPolkit(t *testing.T) {
	if os.Getenv("QUATRRO_POLKIT_TEST") != "1" {
		t.Skip("requires subordinate IDs and namespace support")
	}
	if os.Getenv("QUATRRO_POLKIT_CHILD") != "1" {
		binary, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		policy, err := filepath.Abs("../../packaging/broker/org.quatrro.automations.service.policy")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "unshare", "--user", "--map-auto", "--map-root-user", "--mount", "--pid", "--fork", "--kill-child", "--net", "bwrap",
			"--unshare-pid", "--tmpfs", "/", "--ro-bind", "/usr", "/usr",
			"--symlink", "usr/bin", "/bin", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib", "/lib64",
			"--dir", "/etc", "--ro-bind", "/etc/passwd", "/etc/passwd", "--ro-bind", "/etc/group", "/etc/group",
			"--ro-bind", "/etc/nsswitch.conf", "/etc/nsswitch.conf", "--ro-bind", "/etc/machine-id", "/etc/machine-id",
			"--proc", "/proc", "--dev", "/dev", "--tmpfs", "/run", "--tmpfs", "/tmp", "--chdir", "/",
			"--tmpfs", "/etc/polkit-1", "--tmpfs", "/usr/share/polkit-1/actions", "--tmpfs", "/usr/share/polkit-1/rules.d", "--tmpfs", "/usr/local/share",
			"--ro-bind", policy, "/usr/share/polkit-1/actions/org.quatrro.automations.service.policy",
			"--ro-bind", binary, "/tmp/broker-test", "--setenv", "QUATRRO_POLKIT_CHILD", "1",
			"/tmp/broker-test", "-test.run=^TestIsolatedPolkit$", "-test.v", "-test.timeout=30s")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("isolated Polkit: %v\n%s", err, out)
		}
		t.Log(string(out))
		return
	}
	mapping, mapErr := os.ReadFile("/proc/self/uid_map")
	fields := strings.Fields(string(mapping))
	if mapErr != nil || len(fields) < 3 || fields[0] != "0" || fields[1] == "0" {
		t.Fatal("refusing to run outside subordinate user namespace")
	}
	if os.Getuid() != 0 {
		t.Fatal("namespace root required")
	}
	for _, dir := range []string{"/run/dbus", "/etc/polkit-1/rules.d", "/run/polkit-1/rules.d", "/usr/local/share/polkit-1/rules.d"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	testIsolatedRootPolicy(t)
	config := `<busconfig><type>system</type><listen>unix:path=/run/dbus/system_bus_socket</listen><auth>EXTERNAL</auth><policy context="default"><allow user="*"/><allow own="*"/><allow send_destination="*"/><allow receive_sender="*"/></policy></busconfig>`
	if err := os.WriteFile("/tmp/bus.conf", []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	start := func(name string, args ...string) *exec.Cmd {
		t.Helper()
		c := exec.Command(name, args...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = c.Process.Kill(); _ = c.Wait() })
		return c
	}
	start("/usr/bin/dbus-daemon", "--nofork", "--config-file=/tmp/bus.conf")
	wait := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if check() {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatal("private service readiness timeout")
	}
	wait(func() bool { _, err := os.Stat("/run/dbus/system_bus_socket"); return err == nil })
	rules, err := RenderPolkitRules(Policy{Version: 1, Rules: []Rule{{UID: 1000, Unit: "quatrro-fixture.service", Operations: []string{"status"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile("/etc/polkit-1/rules.d/00-quatrro.rules", rules, 0644); err != nil {
		t.Fatal(err)
	}
	start("/usr/lib/polkit-1/polkitd", "--log-level=debug")
	wait(func() bool {
		return exec.Command("/usr/bin/busctl", "--system", "introspect", "org.freedesktop.PolicyKit1", "/org/freedesktop/PolicyKit1/Authority").Run() == nil
	})
	newSubject := func(uid uint32) string {
		t.Helper()
		subject := start("/usr/bin/setpriv", "--reuid="+strconv.FormatUint(uint64(uid), 10), "--regid="+strconv.FormatUint(uint64(uid), 10), "--clear-groups", "/usr/bin/sleep", "20")
		var since uint64
		wait(func() bool {
			data, e := os.ReadFile(fmt.Sprintf("/proc/%d/stat", subject.Process.Pid))
			if e != nil {
				return false
			}
			since, e = processStartTime(data, subject.Process.Pid)
			status, e2 := os.ReadFile(fmt.Sprintf("/proc/%d/status", subject.Process.Pid))
			return e == nil && e2 == nil && processUIDMatches(status, uid)
		})
		return strconv.Itoa(subject.Process.Pid) + "," + strconv.FormatUint(since, 10) + "," + strconv.FormatUint(uint64(uid), 10)
	}
	allowedSubject := newSubject(1000)
	otherSubject := newSubject(102)
	subject := allowedSubject
	probe := func(unit, op string) int {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		c := exec.CommandContext(ctx, "/usr/bin/pkcheck", "--action-id", "org.quatrro.automations.service."+op, "--process", subject, "--detail", "unit", unit)
		_, e := c.CombinedOutput()
		code := 0
		if e != nil {
			if x, ok := e.(*exec.ExitError); ok {
				code = x.ExitCode()
			} else {
				t.Fatal(e)
			}
		}
		return code
	}
	check := func(unit, op string, want int) {
		t.Helper()
		if code := probe(unit, op); code != want {
			t.Fatalf("%s %s: exit %d want %d", unit, op, code, want)
		}
	}
	check("quatrro-fixture.service", "status", 0)
	subject = otherSubject
	check("quatrro-fixture.service", "status", 1)
	subject = allowedSubject
	check("other.service", "status", 1)
	check("quatrro-fixture.service", "restart", 1)
	empty, err := RenderPolkitRules(Policy{Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile("/etc/polkit-1/rules.d/replacement", empty, 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename("/etc/polkit-1/rules.d/replacement", "/etc/polkit-1/rules.d/00-quatrro.rules"); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { return probe("quatrro-fixture.service", "status") == 1 })
	check("quatrro-fixture.service", "status", 1)
}

// Exercises the production root-only loader, not its UID-injectable test helper.
func testIsolatedRootPolicy(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll("/etc/quatrro", 0755); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"version":1,"rules":[{"uid":1000,"unit":"quatrro-fixture.service","operations":["status"]}]}`)
	if err := os.WriteFile(DefaultPolicyPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if p, err := LoadPolicy(DefaultPolicyPath); err != nil || len(p.Rules) != 1 {
		t.Fatalf("root policy: %v", err)
	}
	if err := os.Chown(DefaultPolicyPath, 1000, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPolicy(DefaultPolicyPath); err != ErrPolicy {
		t.Fatal("non-root owner accepted")
	}
	if err := os.Chown(DefaultPolicyPath, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(DefaultPolicyPath, 0660); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPolicy(DefaultPolicyPath); err != ErrPolicy {
		t.Fatal("group-writable policy accepted")
	}
	if err := os.Chmod(DefaultPolicyPath, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/etc/quatrro/replacement", []byte(`{"version":1,"rules":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename("/etc/quatrro/replacement", DefaultPolicyPath); err != nil {
		t.Fatal(err)
	}
	if p, err := LoadPolicy(DefaultPolicyPath); err != nil || len(p.Rules) != 0 {
		t.Fatal("replacement not observed")
	}
}
