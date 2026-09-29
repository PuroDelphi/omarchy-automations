// Package adapters defines the external, isolated data transformation protocol.
package adapters

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

const Version = 1
const MaxInput = 262144
const MaxOutput = 65536

var identifier = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)

type Manifest struct {
	ID             string   `json:"id"`
	Protocol       int      `json:"protocol"`
	Runtime        string   `json:"runtime"`
	Capabilities   []string `json:"capabilities"`
	InputLimit     int      `json:"input_limit"`
	OutputLimit    int      `json:"output_limit"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

func (m Manifest) Validate() error {
	if !identifier.MatchString(m.ID) || m.Protocol != Version || m.Runtime != "python3" {
		return errors.New("unsupported adapter identity, protocol or runtime")
	}
	if len(m.Capabilities) != 2 {
		return errors.New("adapter must declare event.read and data.write")
	}
	found := map[string]bool{}
	for _, capability := range m.Capabilities {
		if found[capability] || (capability != "event.read" && capability != "data.write") {
			return errors.New("unsupported or duplicate adapter capability")
		}
		found[capability] = true
	}
	if m.InputLimit < 1 || m.InputLimit > MaxInput || m.OutputLimit < 1 || m.OutputLimit > MaxOutput || m.TimeoutSeconds < 1 || m.TimeoutSeconds > 30 {
		return errors.New("adapter quota outside supported bounds")
	}
	return nil
}

type Event struct {
	Source string         `json:"source"`
	Type   string         `json:"type"`
	Data   map[string]any `json:"data"`
}
type Request struct {
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Operation string `json:"operation"`
	Event     Event  `json:"event"`
}
type Response struct {
	Version int            `json:"version"`
	ID      string         `json:"id"`
	Data    map[string]any `json:"data"`
}

func EncodeRequest(m Manifest, id string, event Event) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if !identifier.MatchString(id) {
		return nil, errors.New("invalid adapter invocation ID")
	}
	if event.Data == nil {
		event.Data = map[string]any{}
	}
	data, err := json.Marshal(Request{Version: Version, ID: id, Operation: "transform", Event: event})
	if err != nil || len(data)+1 > m.InputLimit {
		return nil, errors.New("adapter input exceeds quota or is not JSON")
	}
	return append(data, '\n'), nil
}

func DecodeResponse(m Manifest, id string, data []byte) (map[string]any, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if len(data) > m.OutputLimit {
		return nil, errors.New("adapter output exceeds quota")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var response Response
	if err := decoder.Decode(&response); err != nil {
		return nil, errors.New("invalid adapter response")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || response.Version != Version || response.ID != id || response.Data == nil {
		return nil, errors.New("adapter response version, identity or data invalid")
	}
	nodes := 0
	var visit func(any, int) bool
	visit = func(value any, depth int) bool {
		nodes++
		if nodes > 2048 || depth > 16 {
			return false
		}
		switch value := value.(type) {
		case map[string]any:
			for key, v := range value {
				if len(key) > 256 || !visit(v, depth+1) {
					return false
				}
			}
		case []any:
			for _, v := range value {
				if !visit(v, depth+1) {
					return false
				}
			}
		}
		return true
	}
	if !visit(response.Data, 0) {
		return nil, errors.New("adapter output structure exceeds limits")
	}
	return response.Data, nil
}
