package core

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func checkSandboxLimits(path string) error {
	for name, want := range map[string]string{"memory.max": "268435456", "memory.swap.max": "0", "pids.max": "32"} {
		raw, err := os.ReadFile(filepath.Join(path, name))
		if err != nil || strings.TrimSpace(string(raw)) != want {
			return errors.New("required cgroup limit unavailable: " + name)
		}
	}
	raw, err := os.ReadFile(filepath.Join(path, "cpu.max"))
	if err != nil {
		return errors.New("required CPU controller unavailable")
	}
	fields := strings.Fields(string(raw))
	if len(fields) != 2 {
		return errors.New("invalid CPU controller limit")
	}
	quota, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return errors.New("CPU quota is unlimited")
	}
	period, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || period <= 0 || quota <= 0 || quota > period || quota*2 != period {
		return errors.New("required CPU quota is not applied")
	}
	return nil
}

func verifyCommandSandbox(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	unit := "quatrro-probe-" + newID()
	done := make(chan error, 1)
	go func() {
		done <- runIsolatedCommand(ctx, Action{Kind: "command", Executable: "/usr/bin/sleep", Args: []string{"1"}, Timeout: 3}, unit)
	}()
	// Always join cleanup before returning so activation cannot orphan a probe.
	var result error
	checked := false
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if result != nil {
				return result
			}
			if err != nil || !checked {
				return errors.New("sandbox probe failed before verifying kernel controls")
			}
			return nil
		case <-ctx.Done():
			if result == nil {
				result = errors.New("sandbox probe timed out")
			}
			<-done
			return result
		case <-ticker.C:
			if checked || result != nil {
				continue
			}
			commandCtx, commandCancel := context.WithTimeout(ctx, 500*time.Millisecond)
			raw, err := exec.CommandContext(commandCtx, "systemctl", "--user", "show", "--value", "--property=ControlGroup", unit+".service").Output()
			commandCancel()
			group := strings.TrimSpace(string(raw))
			if err != nil || group == "" {
				continue
			}
			if !filepath.IsAbs(group) || group == "/" || filepath.Clean(group) != group || len(group) > 4096 {
				result = errors.New("invalid sandbox cgroup")
				cancel()
				continue
			}
			if err = checkSandboxLimits(filepath.Join("/sys/fs/cgroup", group)); err != nil {
				result = err
				cancel()
				continue
			}
			checked = true
		}
	}
}
