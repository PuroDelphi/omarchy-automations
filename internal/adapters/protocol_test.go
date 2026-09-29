package adapters

import (
	"encoding/json"
	"strings"
	"testing"
)

func manifest() Manifest {
	return Manifest{ID: "example", Protocol: 1, Runtime: "python3", Capabilities: []string{"event.read", "data.write"}, InputLimit: MaxInput, OutputLimit: MaxOutput, TimeoutSeconds: 5}
}
func TestAdapterProtocolQuotasAndCapabilities(t *testing.T) {
	m := manifest()
	input, err := EncodeRequest(m, "run-1", Event{Source: "local:test", Data: map[string]any{"message": "input"}})
	if err != nil {
		t.Fatal(err)
	}
	var request Request
	if err = json.Unmarshal(input, &request); err != nil || request.Operation != "transform" {
		t.Fatal(request, err)
	}
	output, err := DecodeResponse(m, "run-1", []byte(`{"version":1,"id":"run-1","data":{"result":true}}`))
	if err != nil || output["result"] != true {
		t.Fatal(output, err)
	}
	for _, raw := range []string{
		`{"version":2,"id":"run-1","data":{}}`,
		`{"version":1,"id":"other","data":{}}`,
		`{"version":1,"id":"run-1","data":null}`,
		`{"version":1,"id":"run-1","data":{},"command":"run"}`,
		`{"version":1,"id":"run-1","data":{}} {}`,
		`{"version":1,"id":"run-1","data":{"deep":` + strings.Repeat("[", 17) + `0` + strings.Repeat("]", 17) + `}}`,
		strings.Repeat("x", MaxOutput+1),
	} {
		if _, err := DecodeResponse(m, "run-1", []byte(raw)); err == nil {
			t.Fatal("invalid response accepted", len(raw))
		}
	}
	for _, caps := range [][]string{{"event.read", "event.read"}, {"event.read", "system.execute"}, {"event.read"}} {
		bad := m
		bad.Capabilities = caps
		if bad.Validate() == nil {
			t.Fatal("unsupported capability accepted", caps)
		}
	}
	m.InputLimit = len(input) - 1
	if _, err := EncodeRequest(m, "run-1", request.Event); err == nil {
		t.Fatal("newline excluded from quota")
	}
}
