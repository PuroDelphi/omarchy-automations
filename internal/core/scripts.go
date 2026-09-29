package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

func selectedScript(c Config, a Action) (Script, error) {
	for _, s := range c.Scripts {
		if s.ID == a.Script && s.Revision == a.ScriptRevision {
			return s, s.Validate()
		}
	}
	return Script{}, errors.New("script action requires a registered exact revision")
}

func validateScriptAction(c Config, a Action) error {
	s, err := selectedScript(c, a)
	if err != nil {
		return err
	}
	if a.Timeout < 1 || a.Timeout > 300 {
		return errors.New("script timeout must be 1..300 seconds")
	}
	if len(a.ScriptValues)+len(a.ScriptBindings) != len(s.Parameters) {
		return errors.New("script action must supply every approved parameter exactly once")
	}
	for _, p := range s.Parameters {
		value, literal := a.ScriptValues[p.Name]
		path, bound := a.ScriptBindings[p.Name]
		if literal == bound {
			return errors.New("script parameter must have one literal or event binding")
		}
		if literal {
			if _, err := scriptArgument(p, value); err != nil {
				return err
			}
		} else if len(path) > 128 || strings.Count(path, ".") > 8 || !regexp.MustCompile(`^(data\.[a-zA-Z0-9_.-]+|source|type)$`).MatchString(path) {
			return errors.New("invalid script event binding")
		}
	}
	return nil
}

func resolveScriptArguments(c Config, a Action, event Event) ([]string, error) {
	if err := validateScriptAction(c, a); err != nil {
		return nil, err
	}
	s, _ := selectedScript(c, a)
	values := make(map[string]any, len(s.Parameters))
	for name, value := range a.ScriptValues {
		values[name] = value
	}
	for name, path := range a.ScriptBindings {
		value, ok := field(map[string]any{"data": event.Data, "source": event.Source, "type": event.Type}, path)
		if !ok {
			return nil, errors.New("script event parameter unavailable")
		}
		values[name] = value
	}
	return scriptArguments(s, values)
}

func (e *Engine) performScript(ctx context.Context, c Config, a Action, event Event) error {
	args, err := resolveScriptArguments(c, a, event)
	if err != nil {
		return err
	}
	s, _ := selectedScript(c, a)
	command := Action{Directories: a.Directories, WorkingDirectory: a.WorkingDirectory, Kind: "command", Executable: "/usr/bin/" + s.Interpreter, Timeout: a.Timeout}
	if s.Interpreter == "bash" {
		command.Args = []string{"--noprofile", "--norc", "--", "/quatrro/script"}
	} else {
		command.Args = []string{"-I", "-S", "--", "/quatrro/script"}
	}
	command.Args = append(command.Args, args...)
	return runIsolatedInput(ctx, command, "quatrro-script-"+newID(), s.Code)
}

const maxScriptBytes = 32768

type ScriptParameter struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	MaxLength int      `json:"max_length,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Choices   []string `json:"choices,omitempty"`
	Minimum   *int64   `json:"minimum,omitempty"`
	Maximum   *int64   `json:"maximum,omitempty"`
}

// Source paths are only read while preparing a revision. Execution must consume
// Code from the pinned configuration, never reopen the original source path.
type Script struct {
	ID          string            `json:"id"`
	Interpreter string            `json:"interpreter"`
	Code        string            `json:"code"`
	Parameters  []ScriptParameter `json:"parameters"`
	Revision    string            `json:"revision"`
}

func scriptRevision(s Script) string {
	s.Revision = ""
	return canonicalHash(s)
}

func (s Script) Validate() error {
	if !identifier.MatchString(s.ID) {
		return errors.New("invalid script ID")
	}
	if s.Interpreter != "bash" && s.Interpreter != "python3" {
		return errors.New("script interpreter must be bash or python3")
	}
	if len(s.Code) == 0 || len(s.Code) > maxScriptBytes || !utf8.ValidString(s.Code) || strings.ContainsRune(s.Code, 0) {
		return errors.New("script must be nonempty UTF-8 text up to 32 KiB without NUL")
	}
	if s.Parameters == nil || len(s.Parameters) > 16 {
		return errors.New("script supports at most 16 parameters")
	}
	seen := map[string]bool{}
	for _, p := range s.Parameters {
		if !identifier.MatchString(p.Name) || seen[p.Name] {
			return errors.New("invalid or duplicate script parameter")
		}
		seen[p.Name] = true
		switch p.Type {
		case "string":
			if p.MaxLength < 1 || p.MaxLength > 4096 || len(p.Pattern) > 256 || len(p.Choices) > 32 || p.Minimum != nil || p.Maximum != nil {
				return errors.New("invalid string parameter bounds")
			}
			if _, err := regexp.Compile("^(?:" + p.Pattern + ")$"); err != nil {
				return errors.New("invalid script parameter pattern")
			}
			choices := map[string]bool{}
			for _, v := range p.Choices {
				if choices[v] {
					return errors.New("duplicate parameter choice")
				}
				choices[v] = true
				if _, err := scriptArgument(p, v); err != nil {
					return err
				}
			}
		case "integer":
			if p.Minimum == nil || p.Maximum == nil || *p.Minimum > *p.Maximum || *p.Minimum < -9007199254740991 || *p.Maximum > 9007199254740991 || p.MaxLength != 0 || p.Pattern != "" || len(p.Choices) != 0 {
				return errors.New("integer parameter requires exact minimum/maximum within JSON safe range")
			}
		case "boolean":
			if p.Minimum != nil || p.Maximum != nil || p.MaxLength != 0 || p.Pattern != "" || len(p.Choices) != 0 {
				return errors.New("boolean parameter cannot have other bounds")
			}
		default:
			return errors.New("script parameter type must be string, integer or boolean")
		}
	}
	if s.Revision != scriptRevision(s) {
		return errors.New("script revision does not match code and parameter contract")
	}
	return nil
}

func scriptArgument(p ScriptParameter, value any) (string, error) {
	switch p.Type {
	case "string":
		text, ok := value.(string)
		if !ok || !utf8.ValidString(text) || len(text) > p.MaxLength || strings.ContainsRune(text, 0) {
			return "", errors.New("script string argument violates type or length")
		}
		if p.Pattern != "" {
			pattern, err := regexp.Compile("^(?:" + p.Pattern + ")$")
			if err != nil || !pattern.MatchString(text) {
				return "", errors.New("script string argument violates pattern")
			}
		}
		if len(p.Choices) > 0 {
			found := false
			for _, choice := range p.Choices {
				if choice == text {
					found = true
				}
			}
			if !found {
				return "", errors.New("script argument is not an approved choice")
			}
		}
		return text, nil
	case "integer":
		value, ok := value.(float64)
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value || p.Minimum == nil || p.Maximum == nil || value < float64(*p.Minimum) || value > float64(*p.Maximum) {
			return "", errors.New("script integer argument violates type or bounds")
		}
		return strconv.FormatInt(int64(value), 10), nil
	case "boolean":
		value, ok := value.(bool)
		if !ok {
			return "", errors.New("script boolean argument requires a boolean")
		}
		return strconv.FormatBool(value), nil
	}
	return "", errors.New("unknown parameter type")
}

func scriptArguments(s Script, values map[string]any) ([]string, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if len(values) != len(s.Parameters) {
		return nil, errors.New("script parameters must match the approved contract")
	}
	out := make([]string, 0, len(s.Parameters))
	for _, p := range s.Parameters {
		value, ok := values[p.Name]
		if !ok {
			return nil, fmt.Errorf("missing script parameter: %s", p.Name)
		}
		argument, err := scriptArgument(p, value)
		if err != nil {
			return nil, fmt.Errorf("parameter %s: %w", p.Name, err)
		}
		out = append(out, argument)
	}
	return out, nil
}

func prepareScript(raw []byte) (any, error) {
	var request struct {
		ID          string            `json:"id"`
		Path        string            `json:"path"`
		Interpreter string            `json:"interpreter"`
		Parameters  []ScriptParameter `json:"parameters"`
	}
	if err := decode(raw, &request); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(request.Path) || filepath.Clean(request.Path) != request.Path || strings.ContainsRune(request.Path, 0) {
		return nil, errors.New("script source path must be absolute and canonical")
	}
	fd, err := unix.Openat2(unix.AT_FDCWD, request.Path, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_NONBLOCK | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS})
	if err != nil {
		return nil, errors.New("script source unavailable or contains a symbolic link")
	}
	file := os.NewFile(uintptr(fd), "script-source")
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxScriptBytes {
		return nil, errors.New("script source must be a regular file up to 32 KiB")
	}
	code, err := io.ReadAll(io.LimitReader(file, maxScriptBytes+1))
	if err != nil {
		return nil, errors.New("script source read failed")
	}
	script := Script{ID: request.ID, Interpreter: request.Interpreter, Code: string(code), Parameters: request.Parameters}
	if script.Parameters == nil {
		script.Parameters = []ScriptParameter{}
	}
	script.Revision = scriptRevision(script)
	if err := script.Validate(); err != nil {
		return nil, err
	}
	return script, nil
}
