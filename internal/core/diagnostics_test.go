package core

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestDiagnosticsAllowlistAndExclusiveExport(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	marker := "DO-NOT-EXPORT-private-fixture"
	c := notificationConfig()
	c.Actions[0].Title = marker
	c.Flows[0].Name = marker
	c.Destinations = []Destination{{ID: "sensitive", Name: marker, URL: "https://private.example/" + marker, Method: "POST", Headers: map[string]string{"X-Fixture": marker}}}
	activateTest(t, e, c)
	secret, _ := json.Marshal(map[string]string{"id": "credential", "backend": "file", "value": marker})
	if _, err := e.putSecret(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ingest(ctx, Event{Source: "local:test", Data: map[string]any{"sensitive": marker}}); err != nil {
		t.Fatal(err)
	}
	e.lastError.Store(marker) // A future accidental raw error must not leak.
	report, err := e.diagnostics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(report)
	for _, forbidden := range []string{marker, "private.example", e.paths.Config, "credential", "sensitive", "deploy"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatal("diagnostics leak", forbidden)
		}
	}
	m := report.(map[string]any)
	if m["execution_counts"].(map[string]int)["pending"] != 1 {
		t.Fatal("missing aggregate", m)
	}
	path := filepath.Join(t.TempDir(), "diagnostic.json")
	request, _ := json.Marshal(map[string]string{"path": path})
	if _, err := e.diagnosticsFile(ctx, request); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("diagnostic export is not private")
	}
	if _, err := e.diagnosticsFile(ctx, request); err == nil {
		t.Fatal("overwrote diagnostic")
	}
	exported, _ := os.ReadFile(path)
	if strings.Contains(string(exported), marker) {
		t.Fatal("exported secret")
	}
	link := filepath.Join(t.TempDir(), "link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := writeJSONExport(link, report); err == nil {
		t.Fatal("followed export symlink")
	}
}

func TestQueueInspectionPaginationAndRedaction(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	activateTest(t, e, notificationConfig())
	for range 105 {
		if _, err := e.ingest(ctx, Event{Source: "local:test", Data: map[string]any{"message": "private-event-body"}}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := e.queueInspect(ctx, []byte(`{"limit":100}`))
	if err != nil {
		t.Fatal(err)
	}
	page := first.(map[string]any)
	if page["total"] != 105 || page["has_more"] != true || len(page["items"].([]map[string]any)) != 100 {
		t.Fatal(page)
	}
	last, err := e.queueInspect(ctx, []byte(`{"offset":100,"limit":100}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(last.(map[string]any)["items"].([]map[string]any)) != 5 {
		t.Fatal("wrong second page")
	}
	raw, _ := json.Marshal(first)
	if strings.Contains(string(raw), "private-event-body") {
		t.Fatal("queue exposed payload")
	}
	id := page["items"].([]map[string]any)[0]["id"]
	if _, err := e.db.Exec("UPDATE executions SET state='uncertain' WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	result, err := e.queueInspect(ctx, []byte(`{"state":"uncertain"}`))
	if err != nil || result.(map[string]any)["total"] != 1 {
		t.Fatal(result, err)
	}
	for _, request := range []string{`{"state":"completed"}`, `{"limit":101}`, `{"offset":-1}`, `{"state":"' OR 1=1"}`} {
		if _, err := e.queueInspect(ctx, []byte(request)); err == nil {
			t.Fatal("bad queue query accepted", request)
		}
	}
}

func TestImportRejectsFIFOAndSymlinkWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fifo.json")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readJSONImport(path); err == nil {
		t.Fatal("FIFO imported")
	}
	file := filepath.Join(t.TempDir(), "actual.json")
	if err := os.WriteFile(file, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link.json")
	os.Symlink(file, link)
	if _, err := readJSONImport(link); err == nil {
		t.Fatal("import symlink followed")
	}
	if err := os.WriteFile(file, []byte(strings.Repeat("x", (1<<20)+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readJSONImport(file); err == nil {
		t.Fatal("oversized file imported")
	}
}

func TestCredentialRotationImmediatelyChangesIngress(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	put := func(value string) {
		raw, _ := json.Marshal(map[string]string{"id": "incoming", "backend": "file", "value": value})
		if _, err := e.putSecret(ctx, raw); err != nil {
			t.Fatal(err)
		}
	}
	first, second := "first-public-test-token", "second-public-test-token"
	put(first)
	c := notificationConfig()
	c.Entries = []Entry{{ID: "input", Auth: "bearer", Secret: "incoming", Format: "json", Enabled: true}}
	c.Flows[0].Source = "entry:input"
	activateTest(t, e, c)
	handler := e.IngressHandler()
	request := func(value string, want int) {
		r := httptest.NewRequest("POST", "/hooks/input", strings.NewReader(`{"message":"fixture"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+value)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	request(first, 202)
	put(second)
	request(first, 401)
	request(second, 202)
	if _, err := e.deleteSecret(ctx, []byte(`{"id":"incoming"}`)); err != nil {
		t.Fatal(err)
	}
	request(second, 503)
}
