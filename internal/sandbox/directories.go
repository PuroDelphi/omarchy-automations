// Package sandbox implements the private descriptor-based execution helper.
package sandbox

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// Directory identifies both the approved path and its filesystem object.
// Decimal strings preserve 64-bit inode/device values through QML/JSON.
type Directory struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Access string `json:"access"`
	Device string `json:"device"`
	Inode  string `json:"inode"`
}

var targetPattern = regexp.MustCompile(`^/work/[a-z][a-z0-9._-]{0,63}$`)

func validateLocation(source, target, access string) error {
	if !filepath.IsAbs(source) || filepath.Clean(source) != source || source == "/" || len(source) > 4096 || strings.ContainsRune(source, 0) {
		return errors.New("directory source must be an absolute canonical path other than root")
	}
	// Kernel and session interfaces are never approved as working directories.
	for _, path := range []string{"/proc", "/sys", "/dev", "/run"} {
		if source == path || strings.HasPrefix(source, path+"/") {
			return errors.New("system interface directories cannot be exposed")
		}
	}
	if !targetPattern.MatchString(target) || filepath.Clean(target) != target {
		return errors.New("directory target must be /work/identifier")
	}
	if access != "ro" && access != "rw" {
		return errors.New("directory access must be ro or rw")
	}
	return nil
}

func (d Directory) Validate() error {
	if err := validateLocation(d.Source, d.Target, d.Access); err != nil {
		return err
	}
	for _, value := range []string{d.Device, d.Inode} {
		number, err := strconv.ParseUint(value, 10, 64)
		if err != nil || strconv.FormatUint(number, 10) != value {
			return errors.New("invalid directory identity")
		}
	}
	if d.Inode == "0" {
		return errors.New("invalid directory inode")
	}
	return nil
}

func openDirectory(path string) (*os.File, unix.Stat_t, error) {
	var st unix.Stat_t
	fd, err := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{Flags: unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS})
	if err != nil {
		return nil, st, errors.New("directory unavailable or contains symbolic links")
	}
	f := os.NewFile(uintptr(fd), "approved-directory")
	if err = unix.Fstat(fd, &st); err != nil {
		f.Close()
		return nil, st, errors.New("directory identity unavailable")
	}
	return f, st, nil
}

func PrepareDirectory(source, target, access string) (Directory, error) {
	d := Directory{Source: source, Target: target, Access: access}
	if err := validateLocation(source, target, access); err != nil {
		return Directory{}, err
	}
	f, st, err := openDirectory(source)
	if err != nil {
		return Directory{}, err
	}
	defer f.Close()
	d.Device = strconv.FormatUint(uint64(st.Dev), 10)
	d.Inode = strconv.FormatUint(st.Ino, 10)
	return d, d.Validate()
}

// Open validates identity on the same descriptor subsequently passed to bwrap.
func (d Directory) Open() (*os.File, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	f, st, err := openDirectory(d.Source)
	if err != nil {
		return nil, err
	}
	if strconv.FormatUint(uint64(st.Dev), 10) != d.Device || strconv.FormatUint(st.Ino, 10) != d.Inode {
		f.Close()
		return nil, errors.New("approved directory was replaced; prepare and approve it again")
	}
	return f, nil
}
