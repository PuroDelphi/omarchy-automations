package hooks

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var Names = []string{"theme-set", "font-set", "battery-low", "post-boot", "post-update", "pre-refresh-pacman"}
var managementMu sync.Mutex

type AdapterStatus struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func adapterContent(name string) []byte {
	return []byte("#!/bin/bash\n# Quatrro-owned hook; failures never abort the Omarchy operation.\n/usr/bin/timeout --kill-after=0.2s 1.5s \"$HOME/.local/bin/quatrroctl\" hook " + name + " \"$@\" >/dev/null 2>&1 || :\n")
}

// Manage only owns exact, unmodified adapter files. OpenRoot confines filesystem
// access; symlinks are rejected rather than followed, including event folders.
func Manage(directory, operation, name string) ([]AdapterStatus, error) {
	managementMu.Lock()
	defer managementMu.Unlock()
	if operation != "status" && operation != "install" && operation != "remove" {
		return nil, fmt.Errorf("unsupported hook operation")
	}
	valid := false
	for _, n := range Names {
		if n == name {
			valid = true
		}
	}
	if operation != "status" && !valid {
		return nil, fmt.Errorf("unsupported Omarchy hook")
	}
	if operation == "install" {
		if err := os.MkdirAll(directory, 0700); err != nil {
			return nil, err
		}
	}
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		result := []AdapterStatus{}
		for _, n := range Names {
			result = append(result, AdapterStatus{n, "missing"})
		}
		return result, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("hook directory must be a real directory")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	result := []AdapterStatus{}
	for _, n := range Names {
		folder := n + ".d"
		path := filepath.Join(folder, "quatrro-"+n)
		state := "missing"
		folderInfo, folderErr := root.Lstat(folder)
		if folderErr != nil && !os.IsNotExist(folderErr) {
			return nil, folderErr
		}
		if folderErr == nil && (!folderInfo.IsDir() || folderInfo.Mode()&os.ModeSymlink != 0) {
			state = "conflict"
		}
		if state != "conflict" && folderErr == nil {
			info, err := root.Lstat(path)
			if err == nil {
				if !info.Mode().IsRegular() || info.Size() != int64(len(adapterContent(n))) {
					state = "conflict"
				} else {
					content, err := root.ReadFile(path)
					if err != nil {
						return nil, err
					}
					if bytes.Equal(content, adapterContent(n)) && info.Mode().Perm()&0111 != 0 {
						state = "installed"
					} else {
						state = "conflict"
					}
				}
			} else if !os.IsNotExist(err) {
				return nil, err
			}
		}
		if n == name && operation != "status" {
			if state == "conflict" {
				return nil, fmt.Errorf("hook adapter has local changes or an unsafe path; review it manually")
			}
			if operation == "install" && state == "missing" {
				if folderErr != nil {
					if err := root.Mkdir(folder, 0700); err != nil {
						return nil, err
					}
				}
				f, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
				if err != nil {
					return nil, err
				}
				_, err = f.Write(adapterContent(n))
				closeErr := f.Close()
				if err != nil {
					_ = root.Remove(path)
					return nil, err
				}
				if closeErr != nil {
					_ = root.Remove(path)
					return nil, closeErr
				}
				state = "installed"
			}
			if operation == "remove" && state == "installed" {
				if err := root.Remove(path); err != nil {
					return nil, err
				}
				state = "missing"
			}
		}
		result = append(result, AdapterStatus{n, state})
	}
	return result, nil
}
