package sandbox

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryIdentityAndUnsafePaths(t *testing.T) {
	parent := t.TempDir()
	source := filepath.Join(parent, "data")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	d, err := PrepareDirectory(source, "/work/data", "ro")
	if err != nil {
		t.Fatal(err)
	}
	f, err := d.Open()
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err = os.Rename(source, source+"-old"); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if f, err = d.Open(); err == nil {
		f.Close()
		t.Fatal("replacement accepted")
	}
	link := filepath.Join(parent, "link")
	if err = os.Symlink(source, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative", "/", source + "/../data", link, "/run/user", "/proc/self/fd"} {
		if _, err := PrepareDirectory(path, "/work/data", "ro"); err == nil {
			t.Fatal("unsafe path accepted", path)
		}
	}
	for _, target := range []string{"/usr", "/work/..", "/work/a/child", "/quatrro/script"} {
		if _, err := PrepareDirectory(source, target, "rw"); err == nil {
			t.Fatal("unsafe target accepted", target)
		}
	}
	if _, err := PrepareDirectory(source, "/work/data", ""); err == nil {
		t.Fatal("implicit access accepted")
	}
}

func TestRunnerRejectsMalformedTransportAndLimits(t *testing.T) {
	for _, input := range []string{"{}", `{"version":2}`, `{"version":1,"unknown":true}`, `{} {}`, strings.Repeat("x", 1048577)} {
		if Run(strings.NewReader(input)) == nil {
			t.Fatal("invalid request accepted")
		}
	}
	r := Request{Version: 1, Executable: "/usr/bin/true", WorkingDirectory: "/home"}
	if r.Validate() == nil {
		t.Fatal("unapproved cwd accepted")
	}
	r.WorkingDirectory = ""
	r.Args = []string{"bad\x00argument"}
	if r.Validate() == nil {
		t.Fatal("NUL accepted")
	}
}

func TestHostPinnedDirectoryMounts(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("host test")
	}
	for _, access := range []string{"ro", "rw"} {
		t.Run(access, func(t *testing.T) {
			parent := t.TempDir()
			source := filepath.Join(parent, "data")
			if err := os.Mkdir(source, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, "value"), []byte("approved"), 0600); err != nil {
				t.Fatal(err)
			}
			d, err := PrepareDirectory(source, "/work/data", access)
			if err != nil {
				t.Fatal(err)
			}
			code := "[ \"$1\" = '$(touch /tmp/injected); --flag' ] || exit 2\n[ ! -e /tmp/injected ] || exit 3\n[ \"$(cat value)\" = approved ] || exit 4\n[ \"$PWD\" = /work/data ] || exit 5\n"
			if access == "ro" {
				code += "! printf bad > result 2>/dev/null || exit 6\n"
			} else {
				code += "printf output > result || exit 6\n"
			}
			code += "! printf bad >> /quatrro/script 2>/dev/null || exit 7\n"
			r := Request{Version: 1, Executable: "/usr/bin/bash", Args: []string{"--noprofile", "--norc", "--", "/quatrro/script", "$(touch /tmp/injected); --flag"}, Code: code, Directories: []Directory{d}, WorkingDirectory: d.Target}
			cmd, cleanup, err := Command(r)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			// Swap the pathname after validation, before bwrap consumes the descriptors.
			old := source + "-old"
			if err = os.Rename(source, old); err != nil {
				t.Fatal(err)
			}
			if err = os.Mkdir(source, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(source, "value"), []byte("replacement"), 0600); err != nil {
				t.Fatal(err)
			}
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("sandbox: %v %s", err, output)
			}
			if _, err := os.Stat(filepath.Join(source, "result")); !os.IsNotExist(err) {
				t.Fatal("wrote replacement", err)
			}
			output, err := os.ReadFile(filepath.Join(old, "result"))
			if access == "rw" && (err != nil || string(output) != "output") {
				t.Fatal(string(output), err)
			}
			if access == "ro" && !os.IsNotExist(err) {
				t.Fatal("readonly directory written", err)
			}
			if _, _, err := Command(r); err == nil {
				t.Fatal("subsequent execution accepted replacement")
			}
		})
	}
}

func TestHostAdapterTransportAndOutputLimit(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("host test")
	}
	code := "import json,sys\nr=json.load(sys.stdin)\nprint(json.dumps({'version':1,'id':r['id'],'data':{'message':r['event']['data']['message'].upper()}}))\n"
	request := Request{Version: 1, Executable: "/usr/bin/python3", Args: []string{"-I", "-S", "--", "/quatrro/script"}, Code: code, Input: `{"id":"run-1","event":{"data":{"message":"hello"}}}`, OutputLimit: 1024}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := RunIO(bytes.NewReader(raw), &out); err != nil {
		t.Fatal(err)
	}
	var response struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &response); err != nil || response.Data["message"] != "HELLO" {
		t.Fatal(out.String(), err)
	}
	request.Code = "import sys\nsys.stdout.write('x'*70000)\n"
	request.OutputLimit = 64
	raw, _ = json.Marshal(request)
	out.Reset()
	if err := RunIO(bytes.NewReader(raw), &out); err == nil {
		t.Fatal("oversized output succeeded")
	}
	if out.Len() > 64 {
		t.Fatal("output escaped quota", out.Len())
	}
	request.OutputLimit = 0
	request.Code = "print('discarded')\n"
	raw, _ = json.Marshal(request)
	out.Reset()
	if err := RunIO(bytes.NewReader(raw), &out); err != nil || out.Len() != 0 {
		t.Fatal(out.Len(), err)
	}
}
