package main

import (
	"bytes"
	"encoding/json"
	"io"

	"quatrro.local/automations/internal/local"
	"quatrro.local/automations/internal/release"
)

// UI requests stay on stdin. Reject incompatible or ambiguous envelopes before
// resolving paths, connecting to the engine or forwarding any operation.
func decodeUI(raw []byte) (string, json.RawMessage, error) {
	fail := local.ErrIncompatible
	if len(raw) > local.MaxMessage {
		return "", nil, fail
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return "", nil, fail
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return "", nil, fail
		}
		name, ok := token.(string)
		if !ok || fields[name] != nil || (name != "contract" && name != "operation" && name != "data") {
			return "", nil, fail
		}
		var value json.RawMessage
		if d.Decode(&value) != nil {
			return "", nil, fail
		}
		fields[name] = value
	}
	end, err := d.Token()
	if err != nil || end != json.Delim('}') || d.Decode(new(any)) != io.EOF {
		return "", nil, fail
	}
	var contract int
	var operation string
	if json.Unmarshal(fields["contract"], &contract) != nil || contract != release.UIContract || json.Unmarshal(fields["operation"], &operation) != nil || operation == "" || len(operation) > 80 || operation == "ui" || operation == "hello" || operation == "hook" {
		return "", nil, fail
	}
	data := fields["data"]
	if len(data) == 0 || !json.Valid(data) {
		return "", nil, fail
	}
	return operation, data, nil
}
