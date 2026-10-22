package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"quatrro.local/automations/internal/hooks"
)

func (e *Engine) manageHooks(operation string, raw json.RawMessage) (any, error) {
	var request struct {
		Name string `json:"name"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &request); err != nil {
			return nil, err
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(home, ".config", "omarchy", "hooks")
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		directory = filepath.Join(base, "omarchy", "hooks")
	}
	// Development profiles must never modify the desktop's actual hooks.
	if profile := os.Getenv("QUATRRO_PROFILE"); profile != "" {
		directory = filepath.Join(profile, "omarchy-hooks")
	}
	return hooks.Manage(directory, operation, request.Name)
}
