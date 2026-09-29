package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNotificationArgumentsAndLimits(t *testing.T) {
	directory := t.TempDir()
	captured := filepath.Join(directory, "arguments.json")
	executable := filepath.Join(directory, "notify-send")
	script := fmt.Sprintf("#!/usr/bin/python3\nimport json,sys\nwith open(%q,'w') as f: json.dump(sys.argv[1:],f)\n", captured)
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	e := testEngine(t)
	action := Action{Kind: "notify", Title: "{{data.title}}", Body: "{{data.body}}"}
	event := Event{Data: map[string]any{"title": `--help <b>Title</b> & "quoted"`, "body": `<b>Body</b> & <a href="https://example.test">link</a>`}}
	if err := e.perform(context.Background(), action, event); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(captured)
	if err != nil {
		t.Fatal(err)
	}
	var args []string
	if err := json.Unmarshal(raw, &args); err != nil {
		t.Fatal(err)
	}
	want := []string{"--app-name=Omarchy Automations", "--", event.Data["title"].(string), `&lt;b&gt;Body&lt;/b&gt; &amp; &lt;a href=&#34;https://example.test&#34;&gt;link&lt;/a&gt;`}
	if !reflect.DeepEqual(args, want) {
		t.Fatal(args)
	}
	for _, sample := range []struct {
		title, body string
		valid       bool
	}{
		{strings.Repeat("t", 200), strings.Repeat("b", 4096), true},
		{strings.Repeat("t", 201), "body", false},
		{"title", strings.Repeat("b", 4097), false},
		{strings.Repeat("é", 101), "body", false},
	} {
		os.Remove(captured)
		err := e.perform(context.Background(), action, Event{Data: map[string]any{"title": sample.title, "body": sample.body}})
		_, statErr := os.Stat(captured)
		if sample.valid {
			if err != nil || statErr != nil {
				t.Fatal(err, statErr)
			}
		} else {
			if err == nil || err.Error() != "notification exceeds limits" || !os.IsNotExist(statErr) {
				t.Fatal("oversize notification dispatched", err, statErr)
			}
		}
	}
}
