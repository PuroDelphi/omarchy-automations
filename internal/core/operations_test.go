package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestImportDoesNotActivateOrGrant(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	activateTest(t, e, c)
	raw, _ := json.Marshal(c)
	if _, err := e.importConfig(ctx, raw); err != nil {
		t.Fatal(err)
	}
	draft, err := e.config(ctx, "draft")
	if err != nil || draft.Flows[0].Enabled {
		t.Fatal("import left flow active", err)
	}
	active, err := e.config(ctx, "active")
	if err != nil || !active.Flows[0].Enabled {
		t.Fatal("import changed active revision")
	}
}
func TestExportNeverOverwritesAndContainsReferencesOnly(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	activateTest(t, e, c)
	value := "super-private-secret-value"
	secret, _ := json.Marshal(map[string]string{"id": "private", "backend": "file", "value": value})
	if _, err := e.putSecret(ctx, secret); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "export.json")
	req, _ := json.Marshal(map[string]string{"path": path})
	if _, err := e.configFile(ctx, req, true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if canonicalHash(decoded) != canonicalHash(c) {
		t.Fatal("export config mismatch")
	}
	if _, err := e.configFile(ctx, req, true); err == nil {
		t.Fatal("overwrote existing export")
	}
}
func TestStatusReflectsPauseAndHistory(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	e.ingest(ctx, Event{Source: "local:test"})
	e.control(ctx, []byte(`{"paused":true,"admission":"retain"}`))
	result, err := e.status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	m := result.(map[string]any)
	if m["paused"] != true || m["counts"].(map[string]int)["pending"] != 1 {
		t.Fatal(result)
	}
}
