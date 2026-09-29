package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"quatrro.local/automations/internal/sandbox"
)

func TestNamedCommandProfileContract(t *testing.T) {
	d := directoryFixture(t)
	a := Action{ID: "create", Kind: "command", CommandProfile: "make-directory", CommandPath: d.Target + "/new folder; literal", Timeout: 5, Directories: []sandbox.Directory{d}}
	resolved, err := resolveCommand(a)
	if err != nil || resolved.Executable != "/usr/bin/mkdir" || !reflect.DeepEqual(resolved.Args, []string{"--mode=0700", "--", a.CommandPath}) {
		t.Fatal(resolved, err)
	}
	for _, change := range []func(*Action){
		func(a *Action) { a.Args = []string{"--parents"} },
		func(a *Action) { a.Executable = "/usr/bin/sh" },
		func(a *Action) { a.CommandPath = "/work/data/../other" },
		func(a *Action) { a.CommandPath = "/work/data/nested/child" },
		func(a *Action) { a.CommandPath = "/etc/file" },
		func(a *Action) { a.CommandPath = "/work/data/{{data.path}}" },
		func(a *Action) { a.CommandProfile = "unknown" },
		func(a *Action) {
			a.Directories = []sandbox.Directory{{Source: d.Source, Target: d.Target, Access: "ro", Device: d.Device, Inode: d.Inode}}
		},
	} {
		copy := a
		change(&copy)
		if _, err := resolveCommand(copy); err == nil {
			t.Fatal("invalid profile accepted", copy)
		}
	}
	c := notificationConfig()
	a.ID = "notice"
	c.Actions = []Action{a}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	original := capabilities(c)[actionScope(c.Flows[0].ID, a.ID)]
	c.Actions[0].CommandPath = d.Target + "/different"
	if capabilities(c)[actionScope(c.Flows[0].ID, a.ID)] == original {
		t.Fatal("path change retained grant")
	}
	e := testEngine(t)
	raw, _ := json.Marshal(c)
	if _, err := e.saveDraft(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	ev, _ := json.Marshal(Event{Source: c.Flows[0].Source})
	report, err := e.simulate(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	step := report.(map[string]any)["matches"].([]map[string]any)[0]["steps"].([]map[string]any)[0]
	if step["executable"] != "/usr/bin/mkdir" {
		t.Fatal(step)
	}
	if _, err := os.Stat(filepath.Join(d.Source, "different")); !os.IsNotExist(err) {
		t.Fatal("simulation had an effect", err)
	}
}

func TestHostNamedCommandProfiles(t *testing.T) {
	hostIsolation(t)
	for _, profile := range []string{"make-directory", "file-exists"} {
		t.Run(profile, func(t *testing.T) {
			d := directoryFixture(t)
			name := "--option ; literal"
			if profile == "file-exists" {
				if err := os.WriteFile(filepath.Join(d.Source, name), []byte("data"), 0600); err != nil {
					t.Fatal(err)
				}
				d.Access = "ro"
			}
			c := notificationConfig()
			c.Actions = []Action{{ID: "notice", Kind: "command", CommandProfile: profile, CommandPath: d.Target + "/" + name, Directories: []sandbox.Directory{d}, Timeout: 5}}
			e := testEngine(t)
			activateTest(t, e, c)
			if _, err := e.ingest(context.Background(), Event{Source: c.Flows[0].Source}); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if err := e.tick(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			var state string
			if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil || state != "completed" {
				t.Fatal(state, err)
			}
			if profile == "make-directory" {
				info, err := os.Stat(filepath.Join(d.Source, name))
				if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
					t.Fatal(info, err)
				}
				if err := e.perform(context.Background(), c.Actions[0], Event{}); err == nil {
					t.Fatal("existing directory silently accepted")
				}
			} else {
				if err := os.Remove(filepath.Join(d.Source, name)); err != nil {
					t.Fatal(err)
				}
				if err := e.perform(context.Background(), c.Actions[0], Event{}); err == nil {
					t.Fatal("missing file accepted")
				}
			}
		})
	}
}
