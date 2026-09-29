package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"time"
)

// Activation checks required capabilities before installing a revision. The
// runner still fails closed if host capabilities change after this check.
func (e *Engine) preflight(ctx context.Context, c Config) error {
	required := map[string]bool{}
	for _, f := range c.Flows {
		if f.Enabled {
			for _, id := range f.Steps {
				required[id] = true
			}
		}
	}
	var adminCtx context.Context
	checkedAdmin := map[string]bool{}
	needsSandbox := false
	scriptInterpreters := map[string]bool{}
	adapterProbes := map[string]Adapter{}
	for _, a := range c.Actions {
		if !required[a.ID] {
			continue
		}
		if a.Kind == "system-service" {
			key := a.Unit + "/" + a.Operation
			if !checkedAdmin[key] {
				if adminCtx == nil {
					var cancelAdmin context.CancelFunc
					adminCtx, cancelAdmin = context.WithTimeout(ctx, 8*time.Second)
					defer cancelAdmin()
				}
				raw, _ := json.Marshal(map[string]string{"unit": a.Unit, "operation": a.Operation})
				check, err := e.checkAdministrative(adminCtx, raw)
				if err != nil || check.(map[string]any)["state"] != "authorized" {
					return errors.New("administrative permission check failed; verify broker availability, policy and Polkit before activation")
				}
				checkedAdmin[key] = true
			}
		}
		if a.Kind == "adapter" {
			adapter, err := selectedAdapter(c, a)
			if err != nil {
				return err
			}
			needsSandbox = true
			adapterProbes[adapter.Revision] = adapter
		}
		if a.Kind == "command" {
			resolved, err := resolveCommand(a)
			if err != nil {
				return err
			}
			a = resolved
			needsSandbox = true
			info, err := os.Stat(a.Executable)
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
				return errors.New("registered command is unavailable or not executable")
			}
		}
		if a.Kind == "script" {
			needsSandbox = true
			s, err := selectedScript(c, a)
			if err != nil {
				return err
			}
			if _, err := os.Stat("/usr/bin/" + s.Interpreter); err != nil {
				return errors.New("registered script interpreter unavailable")
			}
			scriptInterpreters[s.Interpreter] = true
		}
		if a.Kind == "omarchy" {
			if _, err := exec.LookPath("omarchy"); err != nil {
				return errors.New("Omarchy CLI is unavailable")
			}
		}
	}
	if needsSandbox {
		// Exercise the production runner and verify kernel cgroup controls,
		// rather than assuming systemd accepted properties are enforced.
		var err error
		if e.actionRunner != nil {
			err = e.perform(ctx, Action{Kind: "command", Executable: "/usr/bin/true", Timeout: 5}, Event{})
		} else {
			err = verifyCommandSandbox(ctx)
		}
		if err != nil {
			return errors.New("required command sandbox is unavailable; activation refused")
		}
	}
	for _, a := range c.Actions {
		if !required[a.ID] || (len(a.Directories) == 0 && a.WorkingDirectory == "") {
			continue
		}
		for _, d := range a.Directories {
			f, err := d.Open()
			if err != nil {
				return err
			}
			f.Close()
		}
		if e.actionRunner == nil {
			probe := Action{Kind: "command", Executable: "/usr/bin/true", Timeout: 5, Directories: a.Directories, WorkingDirectory: a.WorkingDirectory}
			code := ""
			if a.Kind == "script" {
				script, _ := selectedScript(c, a)
				probe.Executable = "/usr/bin/" + script.Interpreter
				if script.Interpreter == "bash" {
					probe.Args = []string{"--noprofile", "--norc", "--", "/quatrro/script"}
					code = "exit 0\n"
				} else {
					probe.Args = []string{"-I", "-S", "--", "/quatrro/script"}
					code = "pass\n"
				}
			}
			if err := runIsolatedInput(ctx, probe, "quatrro-mount-probe-"+newID(), code); err != nil {
				return errors.New("required directory sandbox unavailable; activation refused")
			}
		}
	}
	if e.actionRunner == nil {
		for _, probe := range adapterProbes {
			probe.Code = "import json,sys\nr=json.load(sys.stdin)\nprint(json.dumps({'version':1,'id':r['id'],'data':{}}))\n"
			probe.Revision = adapterRevision(probe)
			if _, err := executeAdapter(ctx, probe, Event{}); err != nil {
				return errors.New("required adapter protocol sandbox unavailable; activation refused")
			}
		}
	}
	if e.actionRunner == nil {
		for interpreter := range scriptInterpreters {
			code := "exit 0\n"
			if interpreter == "python3" {
				code = "pass\n"
			}
			probe := Script{ID: "sandbox-probe", Interpreter: interpreter, Code: code, Parameters: []ScriptParameter{}}
			probe.Revision = scriptRevision(probe)
			action := Action{Kind: "script", Script: probe.ID, ScriptRevision: probe.Revision, Timeout: 5}
			if err := e.performScript(ctx, Config{Scripts: []Script{probe}}, action, Event{}); err != nil {
				return errors.New("required script input sandbox is unavailable; activation refused")
			}
		}
	}
	return nil
}

func omarchyCommand(operation string) ([]string, error) {
	switch operation {
	case "theme.current":
		return []string{"theme", "current"}, nil
	case "nightlight.status":
		return []string{"toggle", "nightlight", "--status"}, nil
	case "nightlight.toggle":
		return []string{"toggle", "nightlight"}, nil
	case "system.lock":
		return []string{"system", "lock"}, nil
	}
	return nil, errors.New("unsupported Omarchy operation")
}
func performOmarchy(ctx context.Context, operation string) error {
	args, err := omarchyCommand(operation)
	if err != nil {
		return err
	}
	if err = runBounded(ctx, 10*time.Second, "omarchy", args...); err != nil {
		return errors.New("Omarchy operation failed")
	}
	return nil
}
