package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"quatrro.local/automations/internal/adapters"
	"quatrro.local/automations/internal/local"
)

func adapterManifest() adapters.Manifest {
	return adapters.Manifest{ID: "normalizer", Protocol: 1, Runtime: "python3", Capabilities: []string{"event.read", "data.write"}, InputLimit: 262144, OutputLimit: 4096, TimeoutSeconds: 5}
}

func TestAdapterPreparationPinsLocalCodeAndManifest(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	source := filepath.Join(t.TempDir(), "adapter.py")
	code := "import json,sys\nprint('{}')\n"
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	manifest := adapterManifest()
	prepare := func(path string, m adapters.Manifest) (any, error) {
		raw, _ := json.Marshal(map[string]any{"path": path, "manifest": m})
		return e.Handle(ctx, local.Request{Op: "adapters.prepare", Data: raw})
	}
	value, err := prepare(source, manifest)
	if err != nil {
		t.Fatal(err)
	}
	first := value.(Adapter)
	if err := first.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("raise SystemExit(7)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	value, err = prepare(source, manifest)
	if err != nil {
		t.Fatal(err)
	}
	second := value.(Adapter)
	if first.Code != code || first.Revision == second.Revision {
		t.Fatal("source edits changed snapshot or preserved revision")
	}
	corrupt := first
	corrupt.Code = second.Code
	if corrupt.Validate() == nil {
		t.Fatal("modified code retained old revision")
	}
	corrupt = first
	corrupt.OutputLimit = 2048
	if corrupt.Validate() == nil {
		t.Fatal("changed manifest retained old revision")
	}
	corrupt.Revision = adapterRevision(corrupt)
	if err := corrupt.Validate(); err != nil || corrupt.Revision == first.Revision {
		t.Fatal(err)
	}
	link := source + "-link"
	if err := os.Symlink(source, link); err != nil {
		t.Fatal(err)
	}
	if _, err := prepare(link, manifest); err == nil {
		t.Fatal("symbolic source accepted")
	}
	manifest.Protocol = 2
	if _, err := prepare(source, manifest); err == nil {
		t.Fatal("incompatible protocol accepted")
	}
	var grants, executions int
	if err := e.db.QueryRow("SELECT count(*) FROM grants").Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if err := e.db.QueryRow("SELECT count(*) FROM executions").Scan(&executions); err != nil {
		t.Fatal(err)
	}
	if grants != 0 || executions != 0 {
		t.Fatal("preparation granted or executed", grants, executions)
	}
	c := EmptyConfig()
	c.Adapters = []Adapter{first}
	raw, _ := json.Marshal(c)
	if _, err := e.saveDraft(ctx, raw); err != nil {
		t.Fatal(err)
	}
	stored, err := e.config(ctx, "draft")
	if err != nil || len(stored.Adapters) != 1 || stored.Adapters[0].Code != code {
		t.Fatal(stored, err)
	}
	c.Adapters = append(c.Adapters, first)
	if c.Validate() == nil {
		t.Fatal("duplicate adapter ID accepted")
	}
}
