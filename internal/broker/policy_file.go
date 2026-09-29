package broker

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

const DefaultPolicyPath = "/etc/quatrro/broker.json"

// LoadPolicy reads a fresh policy snapshot. Every path component must be owned
// by root and not writable by group/others. Descriptor-relative traversal keeps
// validation and use bound to the same directories and final regular file.
func LoadPolicy(path string) (Policy, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
		return Policy{}, ErrPolicy
	}
	root, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return Policy{}, ErrPolicy
	}
	defer unix.Close(root)
	return loadPolicyAt(root, strings.Split(strings.TrimPrefix(path, "/"), "/"), 0)
}

// owner is fixed to UID 0 in production. Tests can exercise identical descriptor
// checks inside a private temporary tree without creating host root policy files.
func loadPolicyAt(root int, components []string, owner uint32) (Policy, error) {
	if len(components) == 0 || len(components) > 64 {
		return Policy{}, ErrPolicy
	}
	current, err := unix.Dup(root)
	if err != nil {
		return Policy{}, ErrPolicy
	}
	unix.CloseOnExec(current)
	defer func() { unix.Close(current) }()
	for index, name := range components {
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\x00") {
			return Policy{}, ErrPolicy
		}
		var dir unix.Stat_t
		if unix.Fstat(current, &dir) != nil || dir.Uid != owner || dir.Mode&unix.S_IFMT != unix.S_IFDIR || dir.Mode&0022 != 0 {
			return Policy{}, ErrPolicy
		}
		final := index == len(components)-1
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
		if !final {
			flags |= unix.O_DIRECTORY
		}
		next, openErr := unix.Openat(current, name, flags, 0)
		if openErr != nil {
			return Policy{}, ErrPolicy
		}
		if !final {
			unix.Close(current)
			current = next
			continue
		}
		f := os.NewFile(uintptr(next), "broker-policy")
		defer f.Close()
		var stat unix.Stat_t
		if unix.Fstat(next, &stat) != nil || stat.Uid != owner || stat.Mode&unix.S_IFMT != unix.S_IFREG || stat.Mode&0022 != 0 || stat.Size > MaxPolicy {
			return Policy{}, ErrPolicy
		}
		raw, readErr := io.ReadAll(io.LimitReader(f, MaxPolicy+1))
		if readErr != nil || len(raw) > MaxPolicy {
			return Policy{}, ErrPolicy
		}
		return ParsePolicy(raw)
	}
	return Policy{}, ErrPolicy
}
