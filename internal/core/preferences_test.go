package core

import (
	"context"
	"testing"
)

func TestLanguagePreferenceDefaultsPersistsAndValidates(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	value, err := e.preferences(ctx)
	if err != nil || value.(map[string]string)["language"] != "en" {
		t.Fatal(value, err)
	}
	for _, raw := range []string{`{"language":"fr"}`, `{"language":""}`, `{"language":"es","unexpected":true}`} {
		if _, err := e.setPreferences(ctx, []byte(raw)); err == nil {
			t.Fatal("accepted invalid preference", raw)
		}
	}
	if _, err := e.setPreferences(ctx, []byte(`{"language":"es"}`)); err != nil {
		t.Fatal(err)
	}
	p := e.paths
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	value, err = reopened.preferences(ctx)
	if err != nil || value.(map[string]string)["language"] != "es" {
		t.Fatal(value, err)
	}
}
