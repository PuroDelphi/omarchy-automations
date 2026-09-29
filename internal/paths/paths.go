// Package paths resolves the independent backend's XDG locations.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type Paths struct{ Config, State, Runtime string }

func Resolve() (Paths, error) {
	if profile := os.Getenv("QUATRRO_PROFILE"); profile != "" {
		if !filepath.IsAbs(profile) {
			return Paths{}, fmt.Errorf("QUATRRO_PROFILE must be absolute")
		}
		return Paths{filepath.Join(profile, "config"), filepath.Join(profile, "state"), filepath.Join(profile, "runtime")}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	base := func(env, fallback string) string {
		if v := os.Getenv(env); v != "" {
			return v
		}
		return filepath.Join(home, fallback)
	}
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		return Paths{}, fmt.Errorf("XDG_RUNTIME_DIR is required (use a private directory for development)")
	}
	p := Paths{filepath.Join(base("XDG_CONFIG_HOME", ".config"), "quatrro"), filepath.Join(base("XDG_STATE_HOME", ".local/state"), "quatrro"), filepath.Join(runtime, "quatrro")}
	return p, nil
}

func PrivateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !ok || stat.Uid != uint32(os.Getuid()) || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("directory must be owned by current user with mode 0700: %s", path)
	}
	return nil
}

func (p Paths) Prepare() error {
	for _, d := range []string{p.Config, p.State, p.Runtime} {
		if err := PrivateDir(d); err != nil {
			return err
		}
	}
	return nil
}
func (p Paths) Socket() string { return filepath.Join(p.Runtime, "control.sock") }
