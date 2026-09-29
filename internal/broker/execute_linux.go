package broker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var ErrOutcomeUnknown = errors.New("administrative operation outcome unknown; inspect service before retrying")
var ErrStatus = errors.New("administrative service status unavailable")

type Result struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	Unit        string `json:"unit"`
	LoadState   string `json:"load_state,omitempty"`
	ActiveState string `json:"active_state,omitempty"`
	SubState    string `json:"sub_state,omitempty"`
}

type verifiedPeer interface {
	Peer() Peer
	Revalidate() error
}
type commandRunner func(context.Context, time.Duration, string, []string) ([]byte, error)

// Execute accepts only a kernel-derived identity. No request field can supply a
// policy path, executable, timeout, environment or identity provider.
func Execute(ctx context.Context, identity *PeerIdentity, r Request) (Result, error) {
	if identity == nil {
		return Result{}, ErrDenied
	}
	return execute(ctx, identity, r, func() (Policy, error) { return LoadPolicy(DefaultPolicyPath) }, runCommand)
}

func execute(ctx context.Context, identity verifiedPeer, r Request, load func() (Policy, error), run commandRunner) (Result, error) {
	if ctx.Err() != nil || identity.Revalidate() != nil {
		return Result{}, ErrDenied
	}
	policy, err := load()
	if err != nil {
		return Result{}, ErrDenied
	}
	peer := identity.Peer()
	plan, err := policy.Authorize(peer, r)
	if err != nil {
		return Result{}, ErrDenied
	}
	if _, err = run(ctx, 5*time.Second, "/usr/bin/pkcheck", plan.PolkitArgs); err != nil {
		return Result{}, ErrDenied
	}
	// Policy may have been revoked while authorization was in flight. Reload,
	// rather than carrying the initial allowlist across this boundary.
	if ctx.Err() != nil || identity.Revalidate() != nil || identity.Peer() != peer {
		return Result{}, ErrDenied
	}
	policy, err = load()
	if err != nil {
		return Result{}, ErrDenied
	}
	plan, err = policy.Authorize(peer, r)
	if err != nil {
		return Result{}, ErrDenied
	}
	if ctx.Err() != nil || identity.Revalidate() != nil {
		return Result{}, ErrDenied
	}
	if r.CheckOnly {
		return Result{ID: r.ID, State: "authorized", Unit: r.Unit}, nil
	}
	raw, err := run(ctx, 15*time.Second, plan.Executable, plan.Args)
	if err != nil {
		if r.Operation == "status" {
			return Result{}, ErrStatus
		}
		return Result{}, ErrOutcomeUnknown
	}
	result := Result{ID: r.ID, State: "completed", Unit: r.Unit}
	if r.Operation == "status" {
		values, err := parseStatus(raw, r.Unit)
		if err != nil {
			return Result{}, ErrStatus
		}
		result.LoadState = values["LoadState"]
		result.ActiveState = values["ActiveState"]
		result.SubState = values["SubState"]
	}
	return result, nil
}

var statusValue = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func parseStatus(raw []byte, unit string) (map[string]string, error) {
	if len(raw) > 4096 {
		return nil, ErrStatus
	}
	values := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || values[key] != "" {
			return nil, ErrStatus
		}
		switch key {
		case "Id":
			if value != unit {
				return nil, ErrStatus
			}
		case "LoadState", "ActiveState", "SubState":
			if !statusValue.MatchString(value) {
				return nil, ErrStatus
			}
		default:
			return nil, ErrStatus
		}
		values[key] = value
	}
	if len(values) != 4 {
		return nil, ErrStatus
	}
	return values, nil
}

type boundedOutput struct{ data bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.data.Len()+len(p) > 4096 {
		return 0, ErrStatus
	}
	return b.data.Write(p)
}

func runCommand(ctx context.Context, timeout time.Duration, name string, args []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "SYSTEMD_COLORS=0", "SYSTEMD_PAGERSECURE=1"}
	cmd.Dir = "/"
	cmd.Stdin = nil
	cmd.Stderr = io.Discard
	var out boundedOutput
	cmd.Stdout = &out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		return nil, ErrOutcomeUnknown
	}
	return out.data.Bytes(), nil
}
