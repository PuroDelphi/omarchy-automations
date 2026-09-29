package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html"
	"io"
	"os"
	"os/exec"
	"quatrro.local/automations/internal/sandbox"
	"strings"
	"syscall"
	"time"
)

func runBounded(ctx context.Context, timeout time.Duration, name string, args ...string) error {
	return runBoundedInput(ctx, timeout, nil, name, args...)
}
func runBoundedInput(ctx context.Context, timeout time.Duration, input io.Reader, name string, args ...string) error {
	return runBoundedIO(ctx, timeout, input, nil, name, args...)
}
func runBoundedIO(ctx context.Context, timeout time.Duration, input io.Reader, output io.Writer, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C.UTF-8"}
	for _, key := range []string{"XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS", "HOME", "WAYLAND_DISPLAY", "HYPRLAND_INSTANCE_SIGNATURE", "OMARCHY_PATH", "XDG_CURRENT_DESKTOP", "USER"} {
		if value := os.Getenv(key); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = time.Second
	return cmd.Run() // stderr is discarded; stdout is captured only by explicit bounded callers.
}
func (e *Engine) perform(ctx context.Context, a Action, ev Event) error {
	if e.actionRunner != nil {
		return e.actionRunner(ctx, a, ev)
	}
	switch a.Kind {
	case "omarchy":
		return performOmarchy(ctx, a.Operation)
	case "notify":
		title, err := render(a.Title, ev)
		if err != nil {
			return errors.New("notification title field invalid")
		}
		body, err := render(a.Body, ev)
		if err != nil {
			return errors.New("notification body field invalid")
		}
		if len(title) > 200 || len(body) > 4096 {
			return errors.New("notification exceeds limits")
		}
		if err = runBounded(ctx, 5*time.Second, "notify-send", "--app-name=Omarchy Automations", "--", title, html.EscapeString(body)); err != nil {
			return errors.New("notification service unavailable")
		}
		return nil
	case "service":
		args := []string{"--user"}
		if a.Operation == "status" {
			args = append(args, "is-active", a.Unit)
		} else {
			args = append(args, a.Operation, a.Unit)
		}
		if err := runBounded(ctx, 20*time.Second, "systemctl", args...); err != nil {
			return errors.New("service operation failed")
		}
		if a.Operation != "stop" {
			if err := runBounded(ctx, 5*time.Second, "systemctl", "--user", "is-active", a.Unit); err != nil {
				return errors.New("service not active after operation")
			}
		}
		return nil
	case "command":
		resolved, err := resolveCommand(a)
		if err != nil {
			return err
		}
		return runIsolatedCommand(ctx, resolved, "quatrro-job-"+newID())
	}
	return errors.New("unsupported action")
}

// The caller selects only a fresh, private unit name; payloads cannot supply it.
// Kept separate so host tests observe exactly the production execution path.
func runIsolatedCommand(ctx context.Context, a Action, unit string) error {
	return runIsolatedInput(ctx, a, unit, "")
}
func runIsolatedInput(ctx context.Context, a Action, unit, code string) error {
	return runIsolatedTransport(ctx, a, unit, code, nil, nil)
}
func runIsolatedTransport(ctx context.Context, a Action, unit, code string, transport *sandbox.Request, output io.Writer) error {
	args := []string{"--user", "--wait", "--pipe", "--collect", "--quiet", "--expand-environment=no", "--unit=" + unit,
		"--property=MemoryMax=268435456", "--property=MemorySwapMax=0", "--property=OOMPolicy=kill",
		"--property=CPUQuota=50%", "--property=TasksMax=32", "--property=RuntimeMaxSec=" + (time.Duration(a.Timeout) * time.Second).String(),
		"--property=NoNewPrivileges=yes", "--property=KillMode=control-group", "--property=TimeoutStopSec=1s", "--property=SendSIGKILL=yes",
	}
	base := []string{"--", "/usr/bin/bwrap", "--unshare-all", "--die-with-parent", "--new-session", "--ro-bind", "/usr", "/usr",
		"--symlink", "usr/lib", "/lib", "--symlink", "usr/lib", "/lib64", "--symlink", "usr/bin", "/bin", "--proc", "/proc", "--dev", "/dev",
		"--tmpfs", "/tmp", "--tmpfs", "/home", "--chdir", "/tmp", "--clearenv", "--setenv", "PATH", "/usr/bin"}
	var input io.Reader
	if code != "" {
		input = strings.NewReader(code)
		base = append(base, "--dir", "/quatrro", "--ro-bind-data", "0", "/quatrro/script")
	}
	base = append(base, "--", a.Executable)
	base = append(base, a.Args...)
	if transport != nil || len(a.Directories) > 0 || a.WorkingDirectory != "" {
		request := sandbox.Request{Version: 1, Executable: a.Executable, Args: a.Args, Code: code, Directories: a.Directories, WorkingDirectory: a.WorkingDirectory}
		if transport != nil {
			request = *transport
		}
		if err := request.Validate(); err != nil {
			return err
		}
		executable, err := os.Executable()
		if err != nil {
			return errors.New("sandbox helper executable unavailable")
		}
		payload, err := json.Marshal(request)
		if err != nil {
			return err
		}
		input = bytes.NewReader(payload)
		base = []string{"--", executable, "--sandbox-runner"}
	}
	args = append(args, base...)
	err := runBoundedIO(ctx, time.Duration(a.Timeout+5)*time.Second, input, output, "systemd-run", args...)
	cleanup, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if err != nil {
		// Kill the cgroup even if the systemd-run client was cancelled first.
		_ = runBounded(cleanup, 2*time.Second, "systemctl", "--user", "kill", "--signal=KILL", "--kill-whom=all", unit+".service")
	}
	_ = runBounded(cleanup, 2*time.Second, "systemctl", "--user", "stop", unit+".service")
	if err != nil {
		return errors.New("isolated command failed or sandbox unavailable")
	}
	return nil
}
