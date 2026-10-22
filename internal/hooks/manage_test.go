package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManageLifecyclePreservesOtherHooks(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "theme-set.d", "personal")
	os.MkdirAll(filepath.Dir(other), 0700)
	os.WriteFile(other, []byte("personal"), 0700)
	status, err := Manage(dir, "install", "theme-set")
	if err != nil {
		t.Fatal(err)
	}
	if status[0].State != "installed" {
		t.Fatal(status)
	}
	content, err := os.ReadFile(filepath.Join(dir, "theme-set.d", "quatrro-theme-set"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("../../packaging/hooks/quatrro-theme-set")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(expected) {
		t.Fatal("generated adapter differs from packaged contract")
	}
	if _, err = Manage(dir, "install", "theme-set"); err != nil {
		t.Fatal(err)
	}
	if _, err = Manage(dir, "remove", "theme-set"); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(other); err != nil || string(b) != "personal" {
		t.Fatal("other hook changed", err)
	}
}
func TestManageRejectsModifiedFilesAndLinks(t *testing.T) {
	for _, mode := range []string{"modified", "file-link", "folder-link"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			outside := t.TempDir()
			folder := filepath.Join(dir, "theme-set.d")
			if mode == "folder-link" {
				os.Symlink(outside, folder)
			} else {
				os.Mkdir(folder, 0700)
				p := filepath.Join(folder, "quatrro-theme-set")
				if mode == "file-link" {
					os.Symlink(filepath.Join(outside, "target"), p)
				} else {
					os.WriteFile(p, []byte("custom"), 0700)
				}
			}
			for _, op := range []string{"install", "remove"} {
				if _, err := Manage(dir, op, "theme-set"); err == nil {
					t.Fatal("accepted", mode, op)
				}
			}
		})
	}
	if _, err := Manage(t.TempDir(), "install", "../../bad"); err == nil {
		t.Fatal("accepted unknown hook")
	}
}

func TestStatusDoesNotCreateDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	status, err := Manage(dir, "status", "")
	if err != nil || len(status) != 6 {
		t.Fatal(status, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("status created directory")
	}
}
func TestRootSymlinkRejected(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(t.TempDir(), dir); err != nil {
		t.Fatal(err)
	}
	for _, op := range []string{"status", "install", "remove"} {
		if _, err := Manage(dir, op, "theme-set"); err == nil {
			t.Fatal("accepted root symlink", op)
		}
	}
}
