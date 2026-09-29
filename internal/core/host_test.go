package core

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestHostActions(t *testing.T) {
	if os.Getenv("QUATRRO_HOST_TEST") != "1" {
		t.Skip("requires explicit host integration mode")
	}
	e := testEngine(t)
	ctx := context.Background()
	t.Run("metrics", func(t *testing.T) {
		for _, m := range []Monitor{{ID: "ram", Metric: "memory"}, {ID: "disk", Metric: "disk", Path: "/"}} {
			v, err := e.metric(ctx, m)
			if err != nil || v < 0 || v > 100 {
				t.Fatal("invalid metric", m.Metric, v, err)
			}
		}
		cpu := Monitor{ID: "cpu", Metric: "cpu"}
		_, _ = e.metric(ctx, cpu)
		time.Sleep(50 * time.Millisecond)
		v, err := e.metric(ctx, cpu)
		if err != nil || v < 0 || v > 100 {
			t.Fatal("invalid CPU metric", v, err)
		}
	})
	t.Run("keyring_rotation", func(t *testing.T) {
		id := "host-test"
		put := func(value string) {
			b, _ := json.Marshal(map[string]string{"id": id, "backend": "keyring", "value": value})
			if _, err := e.putSecret(ctx, b); err != nil {
				t.Fatal(err)
			}
		}
		put("public-test-value-one")
		defer func() {
			if _, err := e.deleteSecret(ctx, []byte(`{"id":"host-test"}`)); err != nil {
				t.Error(err)
			}
		}()
		put("public-test-value-two")
		value, err := e.secret(ctx, id)
		if err != nil || value != "public-test-value-two" {
			t.Fatal("keyring rotation failed", err)
		}
	})
	t.Run("notification", func(t *testing.T) {
		if err := e.perform(ctx, Action{Kind: "notify", Title: "Quatrro · prueba de integración", Body: "La conexión con las notificaciones de Omarchy funciona."}, Event{}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("omarchy_readonly", func(t *testing.T) {
		if err := e.perform(ctx, Action{Kind: "omarchy", Operation: "theme.current"}, Event{}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("isolated_command", func(t *testing.T) {
		if err := e.perform(ctx, Action{Kind: "command", Executable: "/usr/bin/true", Timeout: 5}, Event{}); err != nil {
			t.Fatal(err)
		}
		if err := e.perform(ctx, Action{Kind: "command", Executable: "/usr/bin/touch", Args: []string{"/usr/quatrro-should-never-exist"}, Timeout: 5}, Event{}); err == nil {
			t.Fatal("read-only filesystem not enforced")
		}
	})
	t.Run("user_service", func(t *testing.T) {
		unit := "quatrro-test-" + newID() + ".service"
		if err := runBounded(ctx, 5*time.Second, "systemd-run", "--user", "--unit="+unit, "--collect", "--", "/usr/bin/sleep", "60"); err != nil {
			t.Fatal(err)
		}
		defer runBounded(ctx, 5*time.Second, "systemctl", "--user", "stop", unit)
		if err := e.perform(ctx, Action{Kind: "service", Unit: unit, Operation: "restart"}, Event{}); err != nil {
			t.Fatal(err)
		}
	})
}
