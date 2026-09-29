package broker

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

type testPeer struct {
	peer   Peer
	checks int
	failAt int
}

func (p *testPeer) Peer() Peer { return p.peer }
func (p *testPeer) Revalidate() error {
	p.checks++
	if p.failAt > 0 && p.checks >= p.failAt {
		return ErrDenied
	}
	return nil
}

func TestBrokerExecutionAuthorizationOrdering(t *testing.T) {
	for _, scenario := range []string{"success", "policy-denied", "polkit-denied", "revoked", "peer-exited", "command-failed"} {
		t.Run(scenario, func(t *testing.T) {
			peer := &testPeer{peer: Peer{PID: 123, UID: 1000, StartTime: 456}}
			if scenario == "peer-exited" {
				peer.failAt = 2
			}
			request := Request{Version: 1, ID: "job", Unit: "backup.service", Operation: "restart"}
			policy := Policy{Version: 1, Rules: []Rule{{UID: 1000, Unit: "backup.service", Operations: []string{"restart"}}}}
			loads := 0
			calls := []string{}
			load := func() (Policy, error) {
				loads++
				if scenario == "policy-denied" || (scenario == "revoked" && loads == 2) {
					return Policy{Version: 1}, nil
				}
				return policy, nil
			}
			run := func(ctx context.Context, timeout time.Duration, path string, args []string) ([]byte, error) {
				calls = append(calls, path)
				if path == "/usr/bin/pkcheck" {
					if timeout != 5*time.Second {
						t.Fatal("authorization timeout")
					}
					if scenario == "polkit-denied" {
						return nil, errors.New("sensitive diagnostic")
					}
				} else {
					if timeout != 15*time.Second {
						t.Fatal("execution timeout")
					}
					if scenario == "command-failed" {
						return nil, errors.New("sensitive diagnostic")
					}
				}
				return nil, nil
			}
			result, err := execute(context.Background(), peer, request, load, run)
			expected := []string{"/usr/bin/pkcheck"}
			if scenario == "policy-denied" {
				expected = []string{}
			}
			if scenario == "success" || scenario == "command-failed" {
				expected = append(expected, "/usr/bin/systemctl")
			}
			if !reflect.DeepEqual(calls, expected) {
				t.Fatalf("calls=%v expected=%v", calls, expected)
			}
			switch scenario {
			case "success":
				if err != nil || result.State != "completed" {
					t.Fatal(err)
				}
			case "command-failed":
				if err != ErrOutcomeUnknown {
					t.Fatal("uncertainty lost")
				}
			default:
				if err != ErrDenied {
					t.Fatal("denial not redacted")
				}
			}
		})
	}
}

func TestBrokerStatusResponseWhitelist(t *testing.T) {
	good := "Id=backup.service\nLoadState=loaded\nActiveState=active\nSubState=running\n"
	if _, err := parseStatus([]byte(good), "backup.service"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{good + "Environment=secret\n", good + "Id=backup.service\n", "Id=other.service\nLoadState=loaded\nActiveState=active\nSubState=running", "Id=backup.service\n", "Id=backup.service\nLoadState=loaded\nActiveState=active\nSubState=<secret>"} {
		if _, err := parseStatus([]byte(raw), "backup.service"); err != ErrStatus {
			t.Fatal("unsafe status accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &testPeer{peer: Peer{PID: 123, UID: 1000, StartTime: 456}}
	if _, err := execute(ctx, p, Request{}, func() (Policy, error) { t.Fatal("cancelled operation read policy"); return Policy{}, nil }, nil); err != ErrDenied {
		t.Fatal(err)
	}
}

func TestBrokerCommandTimeoutAndOutputQuota(t *testing.T) {
	start := time.Now()
	if _, err := runCommand(context.Background(), 30*time.Millisecond, "/usr/bin/sleep", []string{"30"}); err != ErrOutcomeUnknown {
		t.Fatal("timeout was not uncertain")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("command exceeded bounded shutdown")
	}
	if output, err := runCommand(context.Background(), time.Second, "/usr/bin/head", []string{"-c", "5000", "/dev/zero"}); err != ErrOutcomeUnknown || output != nil {
		t.Fatal("oversized output escaped quota")
	}
	var out boundedOutput
	if _, err := io.Copy(&out, strings.NewReader(strings.Repeat("x", 5000))); err == nil || out.data.Len() > 4096 {
		t.Fatal("io.Copy bypassed output quota")
	}
	if output, err := runCommand(context.Background(), time.Second, "/usr/bin/printf", []string{"safe-status"}); err != nil || string(output) != "safe-status" {
		t.Fatal("bounded command failed", err)
	}
}
