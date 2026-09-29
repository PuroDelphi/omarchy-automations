package main

import (
	"bytes"
	"encoding/json"
	"quatrro.local/automations/internal/broker"
	"strings"
	"testing"
)

func TestPreparePolicyNoSideEffects(t *testing.T) {
	var output bytes.Buffer
	if err := preparePolicy(strings.NewReader(`{"version":1,"rules":[]}`), &output); err != nil {
		t.Fatal(err)
	}
	var p broker.PreparedPolicy
	if err := json.Unmarshal(output.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Policy.Version != 1 || len(p.Policy.Rules) != 0 || !strings.Contains(p.PolkitRules, "var grants = [];") {
		t.Fatal("expected default-deny artifact")
	}
}
func TestPreparePolicyRejectsAmbiguity(t *testing.T) {
	for _, raw := range []string{
		`{"version":1,"version":1,"rules":[]}`,
		`{"version":1,"Version":1,"rules":[]}`,
		`{"version":1,"rules":[],"rul\u0065s":[]}`,
		`{"version":1,"rules":[],"rules":[{"uid":1000,"unit":"a.service","operations":["start"]}]}`,
		`{"version":1,"rules":[{"uid":1000,"uid":102,"unit":"a.service","operations":["start"]}]}`,
		`{"version":1,"rules":[],"shell":"true"}`,
		`{"version":1,"rules":[]} {}`,
		strings.Repeat(" ", broker.MaxPolicy+1),
	} {
		var output bytes.Buffer
		if err := preparePolicy(strings.NewReader(raw), &output); err == nil || output.Len() != 0 {
			t.Fatal("invalid policy produced installation data")
		}
	}
}
