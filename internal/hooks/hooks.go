// Package hooks translates the installed Omarchy hook contract into local events.
package hooks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"quatrro.local/automations/internal/local"
)

const Timeout = time.Second

func Payload(name string, args []string) ([]byte, error) {
	data := map[string]any{}
	expected := 0
	switch name {
	case "theme-set", "font-set", "battery-low":
		expected = 1
	case "post-boot", "post-update", "pre-refresh-pacman":
	default:
		return nil, fmt.Errorf("unsupported Omarchy hook")
	}
	if len(args) != expected {
		return nil, fmt.Errorf("hook expects %d arguments", expected)
	}
	if expected == 1 {
		if len(args[0]) == 0 || len(args[0]) > 1024 || !utf8.ValidString(args[0]) {
			return nil, fmt.Errorf("invalid hook argument")
		}
		switch name {
		case "theme-set":
			data["theme"] = args[0]
		case "font-set":
			data["font"] = args[0]
		case "battery-low":
			n, err := strconv.Atoi(args[0])
			if err != nil || n < 0 || n > 100 {
				return nil, fmt.Errorf("invalid battery percentage")
			}
			data["percentage"] = n
		}
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"id": hex.EncodeToString(id[:]), "source": "hook:" + name, "type": "omarchy." + name, "data": data, "depth": 0})
}

// Send never waits longer than one second. Each invocation is a new event;
// hooks do not retry on ambiguous acknowledgement or queue data on disk.
func Send(parent context.Context, socket, name string, args []string) error {
	payload, err := Payload(name, args)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, Timeout)
	defer cancel()
	_, err = local.Call(ctx, socket, local.Request{Op: "emit", Data: payload})
	return err
}
