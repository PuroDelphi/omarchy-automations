package broker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestPolicyFileDescriptorBoundary(t *testing.T) {
	dir := t.TempDir()
	root, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(root)
	child := filepath.Join(dir, "config")
	if err = os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(child, "broker.json")
	valid := []byte(`{"version":1,"rules":[{"uid":1000,"unit":"backup.service","operations":["restart"]}]}`)
	if err = os.WriteFile(path, valid, 0600); err != nil {
		t.Fatal(err)
	}
	read := func() (Policy, error) {
		return loadPolicyAt(root, []string{"config", "broker.json"}, uint32(os.Getuid()))
	}
	if p, err := read(); err != nil || len(p.Rules) != 1 {
		t.Fatal("safe policy rejected", err)
	}
	if _, err = loadPolicyAt(root, []string{"config", "broker.json"}, uint32(os.Getuid()+1)); err != ErrPolicy {
		t.Fatal("wrong owner accepted")
	}
	for _, target := range []string{dir, child, path} {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.Chmod(target, info.Mode().Perm()|0020); err != nil {
			t.Fatal(err)
		}
		if _, err = read(); err != ErrPolicy {
			t.Fatal("writable path accepted", target)
		}
		if err = os.Chmod(target, info.Mode().Perm()); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.Rename(path, path+".real"); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(path+".real", path); err != nil {
		t.Fatal(err)
	}
	if _, err = read(); err != ErrPolicy {
		t.Fatal("symlink policy accepted")
	}
	os.Remove(path)
	os.Rename(path+".real", path)
	if err = os.Rename(child, child+".real"); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(child+".real", child); err != nil {
		t.Fatal(err)
	}
	if _, err = read(); err != ErrPolicy {
		t.Fatal("symlink directory accepted")
	}
	os.Remove(child)
	os.Rename(child+".real", child)
	if err = os.WriteFile(path, []byte(strings.Repeat("x", MaxPolicy+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = read(); err != ErrPolicy {
		t.Fatal("oversize file accepted")
	}
	os.Remove(path)
	if err = unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = read(); err != ErrPolicy {
		t.Fatal("FIFO accepted")
	}
}

func TestPolicyReloadAndUnsafePath(t *testing.T) {
	dir := t.TempDir()
	root, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(root)
	path := filepath.Join(dir, "policy.json")
	first := []byte(`{"version":1,"rules":[{"uid":1000,"unit":"backup.service","operations":["restart"]}]}`)
	if err = os.WriteFile(path, first, 0600); err != nil {
		t.Fatal(err)
	}
	if p, err := loadPolicyAt(root, []string{"policy.json"}, uint32(os.Getuid())); err != nil || len(p.Rules) != 1 {
		t.Fatal(err)
	}
	if err = os.WriteFile(path+".next", []byte(`{"version":1,"rules":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(path+".next", path); err != nil {
		t.Fatal(err)
	}
	if p, err := loadPolicyAt(root, []string{"policy.json"}, uint32(os.Getuid())); err != nil || len(p.Rules) != 0 {
		t.Fatal("stale policy cached")
	}
	for _, path := range []string{"relative.json", "/etc/../etc/broker.json", "/etc//broker.json", "/", path} {
		if _, err = LoadPolicy(path); err != ErrPolicy {
			t.Fatal("unsafe host path accepted", path)
		}
	}
	for _, components := range [][]string{{"..", "policy.json"}, {"."}, {"a/b"}, {""}} {
		if _, err = loadPolicyAt(root, components, uint32(os.Getuid())); err != ErrPolicy {
			t.Fatal("unsafe relative traversal accepted")
		}
	}
}
