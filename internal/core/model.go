package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"quatrro.local/automations/internal/sandbox"
	"regexp"
	"strings"
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)

type Config struct {
	Adapters     []Adapter     `json:"adapters,omitempty"`
	Version      int           `json:"version"`
	Entries      []Entry       `json:"entries"`
	Destinations []Destination `json:"destinations"`
	Actions      []Action      `json:"actions"`
	Flows        []Flow        `json:"flows"`
	Monitors     []Monitor     `json:"monitors"`
	Timers       []Timer       `json:"timers,omitempty"`
	Scripts      []Script      `json:"scripts,omitempty"`
}
type Entry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Auth    string `json:"auth"`
	Secret  string `json:"secret"`
	Format  string `json:"format"`
	Enabled bool   `json:"enabled"`
}
type Destination struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Format       string            `json:"format,omitempty"`
	Auth         string            `json:"auth"`
	Secret       string            `json:"secret,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	PrivateHosts []string          `json:"private_hosts,omitempty"`
}
type Action struct {
	Adapter          string              `json:"adapter,omitempty"`
	AdapterRevision  string              `json:"adapter_revision,omitempty"`
	CommandProfile   string              `json:"command_profile,omitempty"`
	CommandPath      string              `json:"command_path,omitempty"`
	Directories      []sandbox.Directory `json:"directories,omitempty"`
	WorkingDirectory string              `json:"working_directory,omitempty"`
	Script           string              `json:"script,omitempty"`
	ScriptRevision   string              `json:"script_revision,omitempty"`
	ScriptValues     map[string]any      `json:"script_values,omitempty"`
	ScriptBindings   map[string]string   `json:"script_bindings,omitempty"`
	ID               string              `json:"id"`
	Kind             string              `json:"kind"`
	Title            string              `json:"title,omitempty"`
	Body             string              `json:"body,omitempty"`
	Destination      string              `json:"destination,omitempty"`
	Unit             string              `json:"unit,omitempty"`
	Operation        string              `json:"operation,omitempty"`
	Executable       string              `json:"executable,omitempty"`
	Args             []string            `json:"args,omitempty"`
	Timeout          int                 `json:"timeout_seconds,omitempty"`
}
type Condition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}
type Flow struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Source     string      `json:"source"`
	Enabled    bool        `json:"enabled"`
	Conditions []Condition `json:"conditions,omitempty"`
	Steps      []string    `json:"steps"`
}
type Monitor struct {
	Priority         int     `json:"priority,omitempty"`
	Destination      string  `json:"destination,omitempty"`
	ID               string  `json:"id"`
	Metric           string  `json:"metric"`
	Path             string  `json:"path,omitempty"`
	Unit             string  `json:"unit,omitempty"`
	Threshold        float64 `json:"threshold"`
	Recovery         float64 `json:"recovery"`
	Duration         int     `json:"duration_seconds"`
	RecoveryDuration int     `json:"recovery_duration_seconds,omitempty"`
	Interval         int     `json:"interval_seconds"`
	Cooldown         int     `json:"cooldown_seconds"`
	Enabled          bool    `json:"enabled"`
}
type Event struct {
	ID          string         `json:"id"`
	Source      string         `json:"source"`
	Type        string         `json:"type"`
	Data        map[string]any `json:"data"`
	Correlation string         `json:"correlation,omitempty"`
	Depth       int            `json:"depth"`
}

func decode(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return fmt.Errorf("invalid data: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON value")
	}
	return nil
}
func EmptyConfig() Config {
	return Config{Version: 1, Entries: []Entry{}, Destinations: []Destination{}, Actions: []Action{}, Flows: []Flow{}, Monitors: []Monitor{}}
}

func (c Config) Validate() error {
	if c.Version != 1 {
		return errors.New("unsupported config version")
	}
	if len(c.Entries)+len(c.Destinations)+len(c.Actions)+len(c.Flows)+len(c.Monitors)+len(c.Timers)+len(c.Scripts)+len(c.Adapters) > 500 {
		return errors.New("configuration exceeds 500 resources")
	}
	seen := map[string]bool{}
	check := func(kind, id string) error {
		if !identifier.MatchString(id) {
			return fmt.Errorf("invalid %s id: %s", kind, id)
		}
		k := kind + ":" + id
		if seen[k] {
			return fmt.Errorf("duplicate %s", k)
		}
		seen[k] = true
		return nil
	}
	for _, e := range c.Entries {
		if err := check("entry", e.ID); err != nil {
			return err
		}
		if !identifier.MatchString(e.Secret) {
			return fmt.Errorf("entry %s requires a secret reference", e.ID)
		}
		adapter, supported := inboundProvider(e.Auth)
		if !supported {
			return fmt.Errorf("entry %s: invalid auth", e.ID)
		}
		if adapter.Formats != nil && !adapter.Formats[e.Format] {
			return fmt.Errorf("entry %s: format unsupported by provider", e.ID)
		}
		if e.Format != "json" && e.Format != "form" && e.Format != "raw" && e.Format != "xml" && e.Format != "multipart" {
			return fmt.Errorf("entry %s: unsupported format", e.ID)
		}
	}
	for _, d := range c.Destinations {
		if err := check("destination", d.ID); err != nil {
			return err
		}
		u, err := url.Parse(d.URL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
			return fmt.Errorf("destination %s requires HTTPS without userinfo or fragment", d.ID)
		}
		if err := validatePrivateEndpoints(d); err != nil {
			return err
		}
		if d.Method != "POST" && d.Method != "PUT" && d.Method != "PATCH" {
			return fmt.Errorf("destination %s: unsupported method", d.ID)
		}
		if d.Format != "" && d.Format != "json" && d.Format != "form" && d.Format != "raw" {
			return errors.New("invalid destination format")
		}
		if d.Auth != "" && d.Auth != "bearer" && d.Auth != "hmac" && d.Auth != "oauth2-google" {
			return errors.New("invalid destination auth")
		}
		if d.Auth == "oauth2-google" {
			if err := validateGoogleDestination(d); err != nil {
				return err
			}
		}
		if d.Auth != "" && !identifier.MatchString(d.Secret) {
			return errors.New("destination requires secret reference")
		}
		for k, v := range d.Headers {
			key := strings.ToLower(k)
			if key == "authorization" || key == "host" || key == "cookie" || strings.HasPrefix(key, "proxy-") || strings.ContainsAny(k+v, "\r\n") {
				return errors.New("reserved or invalid header")
			}
		}
	}
	if len(c.Adapters) > 8 {
		return errors.New("at most eight adapter revisions per configuration")
	}
	for _, adapter := range c.Adapters {
		if err := check("adapter", adapter.ID); err != nil {
			return err
		}
		if err := adapter.Validate(); err != nil {
			return err
		}
	}
	if len(c.Scripts) > 8 {
		return errors.New("at most eight script revisions per configuration")
	}
	for _, script := range c.Scripts {
		if err := check("script", script.ID); err != nil {
			return err
		}
		if err := script.Validate(); err != nil {
			return err
		}
	}
	for _, a := range c.Actions {
		if err := check("action", a.ID); err != nil {
			return err
		}
		if len(a.Directories) > 0 || a.WorkingDirectory != "" {
			if a.Kind != "command" && a.Kind != "script" {
				return errors.New("directories require a command or script action")
			}
			if err := (sandbox.Request{Version: 1, Executable: "/usr/bin/true", Directories: a.Directories, WorkingDirectory: a.WorkingDirectory}).Validate(); err != nil {
				return err
			}
		}
		if a.Kind != "command" && (a.CommandProfile != "" || a.CommandPath != "") {
			return errors.New("command profile fields require a command action")
		}
		if a.Kind != "adapter" && (a.Adapter != "" || a.AdapterRevision != "") {
			return errors.New("adapter reference requires an adapter action")
		}
		switch a.Kind {
		case "system-service":
			if _, err := administrativeRequest(a, "validation"); err != nil {
				return err
			}
		case "adapter":
			if _, err := selectedAdapter(c, a); err != nil {
				return err
			}
			if a.Executable != "" || len(a.Args) > 0 || a.Timeout != 0 {
				return errors.New("adapter executable and limits come from its manifest")
			}
		case "script":
			if err := validateScriptAction(c, a); err != nil {
				return err
			}
		case "omarchy":
			if _, err := omarchyCommand(a.Operation); err != nil {
				return err
			}
		case "notify":
			if len(a.Title) > 200 || len(a.Body) > 4096 {
				return errors.New("notification too long")
			}
		case "http":
			if !seen["destination:"+a.Destination] {
				return errors.New("unknown action destination")
			}
			raw := false
			for _, d := range c.Destinations {
				if d.ID == a.Destination && d.Format == "raw" {
					raw = true
				}
			}
			if len(a.Body) > 16384 || (!raw && a.Body != "" && !json.Valid([]byte(a.Body))) {
				return fmt.Errorf("action %s: HTTP body must be valid JSON below 16 KiB", a.ID)
			}
		case "service":
			if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.@-]{0,127}\.service$`).MatchString(a.Unit) {
				return errors.New("invalid service unit")
			}
			if a.Operation != "start" && a.Operation != "stop" && a.Operation != "restart" && a.Operation != "status" {
				return errors.New("invalid service operation")
			}
		case "command":
			if _, err := resolveCommand(a); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported action kind: %s", a.Kind)
		}
	}
	if len(c.Monitors) > 64 {
		return errors.New("at most 64 monitors are supported")
	}
	for _, m := range c.Monitors {
		if err := check("monitor", m.ID); err != nil {
			return err
		}
		if m.Interval < 5 || m.Interval > 3600 || m.Duration < 0 || m.Duration > 86400 || m.RecoveryDuration < 0 || m.RecoveryDuration > 86400 || (m.Cooldown < 5 && m.Metric != "journal") {
			return errors.New("invalid monitor timings")
		}
		if err := validateMonitor(m, seen); err != nil {
			return fmt.Errorf("monitor %s: %w", m.ID, err)
		}
	}
	if len(c.Timers) > 64 {
		return errors.New("at most 64 timers are supported")
	}
	for _, timer := range c.Timers {
		if err := check("timer", timer.ID); err != nil {
			return err
		}
		if err := timer.Validate(); err != nil {
			return fmt.Errorf("timer %s: %w", timer.ID, err)
		}
	}
	for _, f := range c.Flows {
		if err := check("flow", f.ID); err != nil {
			return err
		}
		if len(f.Steps) < 1 || len(f.Steps) > 32 || len(f.Conditions) > 32 {
			return errors.New("flow requires 1..32 steps and at most 32 conditions")
		}
		if !strings.HasPrefix(f.Source, "local:") && !strings.HasPrefix(f.Source, "hook:") && !seen[f.Source] {
			return fmt.Errorf("unknown flow source: %s", f.Source)
		}
		for _, id := range f.Steps {
			if !seen["action:"+id] {
				return fmt.Errorf("unknown action: %s", id)
			}
		}
		for _, cond := range f.Conditions {
			switch cond.Value.(type) {
			case string, float64, bool, nil:
			default:
				return errors.New("condition value must be scalar")
			}
			if len(cond.Field) > 128 || strings.Count(cond.Field, ".") > 8 {
				return errors.New("condition path too deep")
			}
			switch cond.Op {
			case "eq", "ne", "gt", "lt", "contains":
			default:
				return errors.New("invalid condition operator")
			}
		}
	}
	return nil
}
