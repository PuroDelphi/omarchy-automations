package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func scriptConfig(code, interpreter string) Config {
	c := notificationConfig()
	s := Script{ID: "local-script", Interpreter: interpreter, Code: code, Parameters: []ScriptParameter{{Name: "message", Type: "string", MaxLength: 100}}}
	s.Revision = scriptRevision(s)
	c.Scripts = []Script{s}
	c.Actions = []Action{{ID: "notice", Kind: "script", Script: s.ID, ScriptRevision: s.Revision, ScriptBindings: map[string]string{"message": "data.message"}, Timeout: 5}}
	return c
}

func TestScriptActionBindsRevisionAndParameterContract(t *testing.T) {
	c := scriptConfig("exit 0\n", "bash")
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	original := capabilities(c)[actionScope(c.Flows[0].ID, "notice")]
	c.Scripts[0].Code = "exit 1\n"
	c.Scripts[0].Revision = scriptRevision(c.Scripts[0])
	if c.Validate() == nil {
		t.Fatal("updated script silently retained old action revision")
	}
	c.Actions[0].ScriptRevision = c.Scripts[0].Revision
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if capabilities(c)[actionScope(c.Flows[0].ID, "notice")] == original {
		t.Fatal("script change reused capability")
	}
	c.Actions[0].ScriptValues = map[string]any{"message": "duplicate"}
	if c.Validate() == nil {
		t.Fatal("duplicate binding accepted")
	}
}

func TestScriptSimulationResolvesArgumentsWithoutExecution(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := scriptConfig("exit 97\n", "bash")
	raw, _ := json.Marshal(c)
	if _, err := e.saveDraft(ctx, raw); err != nil {
		t.Fatal(err)
	}
	literal := "$(touch /tmp/unwanted); --flag"
	raw, _ = json.Marshal(Event{Source: c.Flows[0].Source, Data: map[string]any{"message": literal}})
	result, err := e.simulate(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	report := result.(map[string]any)
	step := report["matches"].([]map[string]any)[0]["steps"].([]map[string]any)[0]
	if report["effects_executed"] != false || step["script_revision"] != c.Scripts[0].Revision || !reflect.DeepEqual(step["arguments"], []string{literal}) {
		t.Fatal(report)
	}
	var count int
	if err := e.db.QueryRow("SELECT count(*) FROM executions").Scan(&count); err != nil || count != 0 {
		t.Fatal("simulation queued execution", count, err)
	}
	raw, _ = json.Marshal(Event{Source: c.Flows[0].Source, Data: map[string]any{"message": true}})
	if _, err := e.simulate(ctx, raw); err == nil {
		t.Fatal("simulation accepted invalid bound parameter")
	}
}

func TestHostApprovedScriptsRunExactCodeWithLiteralParameters(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("host test")
	}
	for _, interpreter := range []string{"bash", "python3"} {
		t.Run(interpreter, func(t *testing.T) {
			code := "[ \"$1\" = '$(touch /tmp/unwanted); --flag' ] || exit 3\n[ ! -e /tmp/unwanted ] || exit 4\n[ ! -e /home/macondo ] || exit 5\n! printf changed >> /quatrro/script 2>/dev/null || exit 6\n"
			if interpreter == "python3" {
				code = "import sys,os,socket\nassert sys.argv[1] == '$(touch /tmp/unwanted); --flag'\nassert not os.path.exists('/tmp/unwanted')\nassert not os.path.exists('/home/macondo')\ntry:\n open('/quatrro/script','w').write('changed')\n raise AssertionError('writable source')\nexcept OSError: pass\n"
			}
			e := testEngine(t)
			c := scriptConfig(code, interpreter)
			source := filepath.Join(t.TempDir(), "original-script")
			if err := os.WriteFile(source, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(map[string]any{"id": c.Scripts[0].ID, "path": source, "interpreter": interpreter, "parameters": c.Scripts[0].Parameters})
			prepared, err := prepareScript(raw)
			if err != nil {
				t.Fatal(err)
			}
			c.Scripts[0] = prepared.(Script)
			c.Actions[0].ScriptRevision = c.Scripts[0].Revision
			activateTest(t, e, c)
			if err := os.WriteFile(source, []byte("invalid replacement code"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := e.ingest(context.Background(), Event{Source: c.Flows[0].Source, Data: map[string]any{"message": "$(touch /tmp/unwanted); --flag"}}); err != nil {
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
		})
	}
}

func TestHostScriptPendingPermissionAndParameterChecks(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("host test")
	}
	for _, scenario := range []string{"revoked", "replaced-revision", "missing", "wrong-type", "too-long"} {
		t.Run(scenario, func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			c := scriptConfig("exit 0\n", "bash")
			activateTest(t, e, c)
			data := map[string]any{"message": "valid"}
			switch scenario {
			case "missing":
				delete(data, "message")
			case "wrong-type":
				data["message"] = true
			case "too-long":
				data["message"] = strings.Repeat("x", 101)
			}
			if _, err := e.ingest(ctx, Event{Source: c.Flows[0].Source, Data: data}); err != nil {
				t.Fatal(err)
			}
			expected := "failed"
			if scenario == "revoked" {
				raw, _ := json.Marshal(map[string]string{"scope": actionScope(c.Flows[0].ID, "notice")})
				if _, err := e.revoke(ctx, raw); err != nil {
					t.Fatal(err)
				}
				expected = "denied"
			} else if scenario == "replaced-revision" {
				c.Scripts[0].Code = "exit 1\n"
				c.Scripts[0].Revision = scriptRevision(c.Scripts[0])
				c.Actions[0].ScriptRevision = c.Scripts[0].Revision
				activateTest(t, e, c)
				expected = "denied"
			}
			for range 2 {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var state string
			if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil || state != expected {
				t.Fatalf("state=%s want=%s error=%v", state, expected, err)
			}
			if expected == "failed" {
				var message string
				if err := e.db.QueryRow("SELECT message FROM steps").Scan(&message); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(message, "parameter") {
					t.Fatalf("failure did not originate in parameter validation: %s", message)
				}
			}
		})
	}
}

func TestPreparedScriptPinsBytesAndRejectsUnsafeSources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "task.sh")
	if err := os.WriteFile(path, []byte("printf '%s' \"$1\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	prepare := func(path string) (any, error) {
		raw, _ := json.Marshal(map[string]any{"id": "task", "path": path, "interpreter": "bash", "parameters": []ScriptParameter{}})
		return prepareScript(raw)
	}
	value, err := prepare(path)
	if err != nil {
		t.Fatal(err)
	}
	first := value.(Script)
	if err := os.WriteFile(path, []byte("exit 2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	value, err = prepare(path)
	if err != nil {
		t.Fatal(err)
	}
	second := value.(Script)
	if first.Revision == second.Revision || first.Code != "printf '%s' \"$1\"\n" {
		t.Fatal("source edit changed approved snapshot")
	}
	first.Code = "changed"
	if first.Validate() == nil {
		t.Fatal("accepted changed content with old revision")
	}
	link := filepath.Join(dir, "link.sh")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{link, dir, "relative.sh"} {
		if _, err := prepare(source); err == nil {
			t.Fatal("accepted unsafe source", source)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxScriptBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepare(path); err == nil {
		t.Fatal("accepted oversized script")
	}
}

func TestScriptParametersPreserveTypesAndLiteralArguments(t *testing.T) {
	minimum, maximum := int64(1), int64(10)
	s := Script{ID: "task", Interpreter: "bash", Code: "printf '%s' \"$1\"", Parameters: []ScriptParameter{{Name: "message", Type: "string", MaxLength: 100}, {Name: "copies", Type: "integer", Minimum: &minimum, Maximum: &maximum}, {Name: "enabled", Type: "boolean"}}}
	s.Revision = scriptRevision(s)
	values := map[string]any{"message": "$(touch /tmp/should-not-run); --flag", "copies": float64(2), "enabled": true}
	args, err := scriptArguments(s, values)
	if err != nil || !reflect.DeepEqual(args, []string{values["message"].(string), "2", "true"}) {
		t.Fatal(args, err)
	}
	for _, v := range []any{"2", float64(1.5), float64(11), nil} {
		values["copies"] = v
		if _, err := scriptArguments(s, values); err == nil {
			t.Fatal("invalid integer accepted", v)
		}
	}
	values["copies"] = float64(2)
	values["extra"] = "unexpected"
	if _, err := scriptArguments(s, values); err == nil {
		t.Fatal("extra parameter accepted")
	}
	delete(values, "extra")
	values["message"] = "bad\x00arg"
	if _, err := scriptArguments(s, values); err == nil {
		t.Fatal("NUL accepted")
	}
	s.Parameters[0].Pattern = "[a-z]+"
	s.Parameters[0].Choices = []string{"safe", "other"}
	s.Revision = scriptRevision(s)
	values["message"] = "safe"
	if _, err := scriptArguments(s, values); err != nil {
		t.Fatal(err)
	}
	values["message"] = "safe;command"
	if _, err := scriptArguments(s, values); err == nil {
		t.Fatal("pattern bypass")
	}
	s.Parameters[0].MaxLength = 0
	s.Revision = scriptRevision(s)
	if s.Validate() == nil {
		t.Fatal("unbounded string")
	}
}
