package sandbox

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

type Request struct {
	Input            string      `json:"input,omitempty"`
	OutputLimit      int         `json:"output_limit,omitempty"`
	Version          int         `json:"version"`
	Executable       string      `json:"executable"`
	Args             []string    `json:"args"`
	Code             string      `json:"code,omitempty"`
	Directories      []Directory `json:"directories,omitempty"`
	WorkingDirectory string      `json:"working_directory,omitempty"`
}

func (r Request) Validate() error {
	if len(r.Input) > 262144 || r.OutputLimit < 0 || r.OutputLimit > 65536 {
		return errors.New("sandbox I/O exceeds limits")
	}
	if r.Version != 1 {
		return errors.New("unsupported sandbox request version")
	}
	if !strings.HasPrefix(r.Executable, "/usr/bin/") || filepath.Clean(r.Executable) != r.Executable || strings.ContainsRune(r.Executable, 0) {
		return errors.New("invalid sandbox executable")
	}
	if len(r.Args) > 64 || len(r.Directories) > 8 || len(r.Code) > 32768 || !utf8.ValidString(r.Code) || strings.ContainsRune(r.Code, 0) {
		return errors.New("sandbox request exceeds limits")
	}
	for _, arg := range r.Args {
		if len(arg) > 4096 || strings.ContainsRune(arg, 0) {
			return errors.New("invalid sandbox argument")
		}
	}
	targets := map[string]bool{}
	for _, d := range r.Directories {
		if err := d.Validate(); err != nil {
			return err
		}
		if targets[d.Target] {
			return errors.New("duplicate sandbox directory target")
		}
		targets[d.Target] = true
	}
	if r.WorkingDirectory != "" && r.WorkingDirectory != "/tmp" && !targets[r.WorkingDirectory] {
		return errors.New("working directory must be an approved mount target")
	}
	return nil
}

// Command prepares pinned descriptors before starting bwrap. Always call cleanup,
// including after a Start failure. Descriptors are not inherited by the payload.
func Command(r Request) (*exec.Cmd, func(), error) {
	if err := r.Validate(); err != nil {
		return nil, nil, err
	}
	files := []*os.File{}
	cleanup := func() {
		for _, f := range files {
			f.Close()
		}
	}
	fail := func(err error) (*exec.Cmd, func(), error) { cleanup(); return nil, nil, err }
	args := []string{"--unshare-all", "--die-with-parent", "--new-session", "--ro-bind", "/usr", "/usr", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib", "/lib64", "--symlink", "usr/bin", "/bin", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--tmpfs", "/home", "--dir", "/work", "--clearenv", "--setenv", "PATH", "/usr/bin"}
	for _, d := range r.Directories {
		f, err := d.Open()
		if err != nil {
			return fail(err)
		}
		files = append(files, f)
		option := "--ro-bind-fd"
		if d.Access == "rw" {
			option = "--bind-fd"
		}
		args = append(args, option, strconv.Itoa(2+len(files)), d.Target)
	}
	if r.Code != "" {
		fd, err := unix.MemfdCreate("quatrro-script", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
		if err != nil {
			return fail(err)
		}
		f := os.NewFile(uintptr(fd), "script-code")
		files = append(files, f)
		if _, err = f.WriteString(r.Code); err != nil {
			return fail(err)
		}
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return fail(err)
		}
		if _, err = unix.FcntlInt(f.Fd(), unix.F_ADD_SEALS, unix.F_SEAL_WRITE|unix.F_SEAL_GROW|unix.F_SEAL_SHRINK|unix.F_SEAL_SEAL); err != nil {
			return fail(err)
		}
		args = append(args, "--dir", "/quatrro", "--ro-bind-data", strconv.Itoa(2+len(files)), "/quatrro/script")
	}
	cwd := r.WorkingDirectory
	if cwd == "" {
		cwd = "/tmp"
	}
	args = append(args, "--chdir", cwd, "--", r.Executable)
	args = append(args, r.Args...)
	cmd := exec.Command("/usr/bin/bwrap", args...)
	cmd.ExtraFiles = files
	cmd.Stdin = strings.NewReader(r.Input)
	cmd.Env = []string{"PATH=/usr/bin", "LANG=C.UTF-8"}
	return cmd, cleanup, nil
}

// Run consumes the private stdin transport. This is not an authorization API:
// the engine must validate capabilities before starting its constrained unit.
func Run(input io.Reader) error { return RunIO(input, io.Discard) }

// cappedOutput bounds bytes forwarded to the parent. A failed/partial response
// must never be used by the caller. Unit timeout still bounds a stuck child.
type cappedOutput struct {
	destination io.Writer
	remaining   int
}

func (w *cappedOutput) Write(data []byte) (int, error) {
	if len(data) > w.remaining {
		return 0, errors.New("sandbox output exceeds quota")
	}
	n, err := w.destination.Write(data)
	w.remaining -= n
	return n, err
}

func RunIO(input io.Reader, output io.Writer) error {
	data, err := io.ReadAll(io.LimitReader(input, 1048577))
	if err != nil || len(data) > 1048576 {
		return errors.New("invalid sandbox request size")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var r Request
	if err = decoder.Decode(&r); err != nil {
		return errors.New("invalid sandbox request")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return errors.New("trailing sandbox request data")
	}
	cmd, cleanup, err := Command(r)
	if err != nil {
		return err
	}
	defer cleanup()
	if r.OutputLimit > 0 {
		cmd.Stdout = &cappedOutput{destination: output, remaining: r.OutputLimit}
	}
	if err = cmd.Run(); err != nil {
		return errors.New("sandbox process failed")
	}
	return nil
}
