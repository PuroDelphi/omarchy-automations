package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestUIEnvelopePreservesPayload(t *testing.T) {
	payload := json.RawMessage(`{"value":"secret\u0020text","nested":[1,true]}`)
	raw, _ := json.Marshal(map[string]any{"contract": 1, "operation": "secrets.put", "data": payload})
	op, data, err := decodeUI(raw)
	if err != nil || op != "secrets.put" || !bytes.Equal(data, payload) {
		t.Fatal(op, string(data), err)
	}
}
func TestUIEnvelopeRejectsIncompatibleOrAmbiguousInput(t *testing.T) {
	for _, raw := range []string{
		`{"contract":2,"operation":"emit","data":{}}`,
		`{"operation":"emit","data":{}}`,
		`{"contract":1,"contract":2,"operation":"emit","data":{}}`,
		`{"contract":1,"Contract":1,"operation":"emit","data":{}}`,
		`{"contract":1,"operation":"emit","operation":"control","data":{}}`,
		`{"contract":1,"operation":"hello","data":{}}`,
		`{"contract":1,"operation":"ui","data":{}}`,
		`{"contract":1,"operation":"hook","data":{}}`,
		`{"contract":1,"operation":"emit","data":{}} {}`,
		`{"contract":1,"operation":"emit"}`,
	} {
		op, data, err := decodeUI([]byte(raw))
		if err == nil || op != "" || data != nil {
			t.Fatal("invalid UI envelope forwarded")
		}
	}
}
