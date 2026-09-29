package broker

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestCheckOnlyNeverInvokesSystemctl(t *testing.T) {
	for _, op := range []string{"status", "start", "stop", "restart"} {
		for _, scenario := range []string{"authorized", "policy-denied", "polkit-denied", "revoked"} {
			t.Run(op+"/"+scenario, func(t *testing.T) {
				r := Request{Version: 1, ID: "probe", Unit: "backup.service", Operation: op, CheckOnly: true}
				p := Policy{Version: 1, Rules: []Rule{{UID: 1000, Unit: r.Unit, Operations: []string{op}}}}
				peer := &testPeer{peer: Peer{PID: 42, UID: 1000, StartTime: 100}}
				loads, calls := 0, 0
				result, err := execute(context.Background(), peer, r, func() (Policy, error) {
					loads++
					if scenario == "policy-denied" || (scenario == "revoked" && loads == 2) {
						return Policy{Version: 1}, nil
					}
					return p, nil
				}, func(_ context.Context, _ time.Duration, name string, _ []string) ([]byte, error) {
					calls++
					if name != "/usr/bin/pkcheck" {
						t.Fatal("preflight attempted an effect")
					}
					if scenario == "polkit-denied" {
						return nil, ErrDenied
					}
					return nil, nil
				})
				if scenario == "authorized" {
					if err != nil || result.State != "authorized" || result.ActiveState != "" {
						t.Fatalf("%+v %v", result, err)
					}
				} else if err != ErrDenied {
					t.Fatal("denial lost")
				}
				if calls > 1 {
					t.Fatal("unexpected retries")
				}
			})
		}
	}
}

func TestCheckResponseCannotMasqueradeAsExecution(t *testing.T) {
	r := Request{Version: 1, ID: "probe", Unit: "backup.service", Operation: "restart", CheckOnly: true}
	raw, _ := json.Marshal(Response{Version: 1, OK: true, Result: &Result{ID: r.ID, Unit: r.Unit, State: "authorized"}})
	if _, err := parseResponse(raw, r); err != nil {
		t.Fatal(err)
	}
	r.CheckOnly = false
	if _, err := parseResponse(raw, r); err != ErrOutcomeUnknown {
		t.Fatal("authorization accepted as executed effect")
	}
	raw, _ = json.Marshal(Response{Version: 1, OK: true, Result: &Result{ID: r.ID, Unit: r.Unit, State: "completed"}})
	r.CheckOnly = true
	if _, err := parseResponse(raw, r); err != ErrOutcomeUnknown {
		t.Fatal("completion accepted as no-effect check")
	}
	raw = []byte(`{"version":1,"ok":false,"error":"invalid_request"}`)
	if _, err := parseResponse(raw, r); err != ErrRequest {
		t.Fatal("legacy rejection not recognized")
	}
}
