package core

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"quatrro.local/automations/internal/local"
)

func exportPath(data []byte) (string, error) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decode(data, &req); err != nil {
		return "", err
	}
	if !filepath.IsAbs(req.Path) || filepath.Ext(req.Path) != ".json" {
		return "", errors.New("choose an absolute .json path")
	}
	return req.Path, nil
}

func writeJSONExport(path string, value any) (any, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(raw) > local.MaxMessage {
		raw, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	if len(raw) > local.MaxMessage {
		return nil, errors.New("export exceeds portable 1 MiB limit")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("cannot create export; choose a new filename")
	}
	defer f.Close()
	if _, err = f.Write(raw); err != nil {
		return nil, errors.New("export write failed; remove incomplete output before retry")
	}
	if err = f.Sync(); err != nil {
		return nil, errors.New("export sync failed; output may be incomplete")
	}
	if err = f.Close(); err != nil {
		return nil, errors.New("export close failed")
	}
	return map[string]string{"path": path}, nil
}

func readJSONImport(path string) ([]byte, error) {
	// Nonblocking open prevents FIFO hangs; fstat/read refer to the same inode.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errors.New("cannot open regular JSON import")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > local.MaxMessage {
		return nil, errors.New("import requires a regular JSON file below 1 MiB")
	}
	raw, err := io.ReadAll(io.LimitReader(f, local.MaxMessage+1))
	if err != nil || len(raw) > local.MaxMessage {
		return nil, errors.New("import exceeds read limit or is unreadable")
	}
	return raw, nil
}
