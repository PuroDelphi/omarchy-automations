package core

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// resolveCommand is shared by validation, preflight, simulation and dispatch.
// Named profiles derive argv; imported or event data cannot add flags.
func resolveCommand(a Action) (Action, error) {
	if a.Timeout < 1 || a.Timeout > 300 {
		return Action{}, errors.New("command timeout must be 1..300 seconds")
	}
	switch a.CommandProfile {
	case "", "fixed":
		if a.CommandPath != "" {
			return Action{}, errors.New("fixed command cannot have a profile path")
		}
		if !filepath.IsAbs(a.Executable) || filepath.Clean(a.Executable) != a.Executable || !strings.HasPrefix(a.Executable, "/usr/bin/") || strings.ContainsRune(a.Executable, 0) || len(a.Args) > 32 {
			return Action{}, errors.New("command needs absolute executable and at most 32 fixed args")
		}
		for _, arg := range a.Args {
			if len(arg) > 4096 || strings.ContainsRune(arg, 0) || strings.Contains(arg, "{{") {
				return Action{}, errors.New("command args must be fixed and bounded")
			}
		}
		return a, nil
	case "file-exists", "make-directory":
		if a.Executable != "" || len(a.Args) > 0 {
			return Action{}, errors.New("named command profile cannot override executable or arguments")
		}
	default:
		return Action{}, errors.New("unsupported command profile")
	}
	path := a.CommandPath
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, 0) || strings.Contains(path, "{{") || !utf8.ValidString(path) || len(filepath.Base(path)) > 255 {
		return Action{}, errors.New("profile path must name one direct child of an approved mount")
	}
	permitted := false
	for _, d := range a.Directories {
		if err := d.Validate(); err != nil {
			return Action{}, err
		}
		if d.Target == filepath.Dir(path) && (a.CommandProfile != "make-directory" || d.Access == "rw") {
			permitted = true
		}
	}
	if !permitted {
		return Action{}, errors.New("profile requires an approved parent mount with appropriate access")
	}
	if a.CommandProfile == "file-exists" {
		a.Executable = "/usr/bin/test"
		a.Args = []string{"-f", path}
	} else {
		a.Executable = "/usr/bin/mkdir"
		a.Args = []string{"--mode=0700", "--", path}
	}
	return a, nil
}
