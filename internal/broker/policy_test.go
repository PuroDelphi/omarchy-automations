package broker

import (
	"strings"
	"testing"
)

func TestAdministrativePolicyBoundary(t *testing.T) {
	p, err := ParsePolicy([]byte(`{"version":1,"rules":[{"uid":1000,"unit":"backup.service","operations":["status","restart"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	peer := Peer{PID: 123, UID: 1000, StartTime: 456}
	r := Request{Version: 1, ID: "job-1", Unit: "backup.service", Operation: "restart"}
	plan, err := p.Authorize(peer, r)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Executable != "/usr/bin/systemctl" || strings.Join(plan.Args, " ") != "--system --no-ask-password --no-pager restart -- backup.service" {
		t.Fatal("unexpected command plan")
	}
	joined := strings.Join(plan.PolkitArgs, " ")
	if !strings.Contains(joined, "--process 123,456,1000") || strings.Contains(joined, "--allow-user-interaction") || !strings.Contains(joined, "--detail unit backup.service") {
		t.Fatal("unsafe authorization subject")
	}
	for _, change := range []Request{{Version: 1, ID: "job", Unit: "other.service", Operation: "restart"}, {Version: 1, ID: "job", Unit: "backup.service", Operation: "start"}, {Version: 1, ID: "job", Unit: "backup.service", Operation: "enable"}, {Version: 2, ID: "job", Unit: "backup.service", Operation: "restart"}, {Version: 1, ID: "job", Unit: "*.service", Operation: "restart"}, {Version: 1, ID: "job", Unit: "../backup.service", Operation: "restart"}, {Version: 1, ID: "job", Unit: "backup@.service", Operation: "restart"}} {
		if _, err = p.Authorize(peer, change); err != ErrDenied {
			t.Fatal("request outside allowlist accepted")
		}
	}
	for _, bad := range []Peer{{123, 0, 456}, {123, 1001, 456}, {0, 1000, 456}, {123, 1000, 0}} {
		if _, err = p.Authorize(bad, r); err != ErrDenied {
			t.Fatal("untrusted peer accepted")
		}
	}
	if _, err = (Policy{Version: 1}).Authorize(peer, r); err != ErrDenied {
		t.Fatal("empty policy granted operation")
	}
}

func TestAdministrativeJSONRejectsPrivilegeInputs(t *testing.T) {
	for _, raw := range []string{
		`{"version":1,"id":"job","unit":"backup.service","operation":"restart","uid":0}`,
		`{"version":1,"id":"job","unit":"backup.service","operation":"restart","args":["--root=/tmp"]}`,
		`{"version":1,"id":"job","unit":"backup.service","operation":"restart","executable":"/bin/sh"}`,
		`{"version":1,"id":"job","unit":"backup.service","operation":"restart"} {}`,
		strings.Repeat("x", MaxRequest+1),
	} {
		if _, err := ParseRequest([]byte(raw)); err != ErrRequest {
			t.Fatal("untrusted privilege input accepted")
		}
	}
	for _, raw := range []string{
		`{"version":1,"rules":[{"uid":0,"unit":"backup.service","operations":["restart"]}]}`,
		`{"version":1,"rules":[{"uid":1000,"unit":"*.service","operations":["restart"]}]}`,
		`{"version":1,"rules":[{"uid":1000,"unit":"backup.service","operations":["restart","restart"]}]}`,
		`{"version":1,"rules":[{"uid":1000,"unit":"backup.service","operations":["restart"],"shell":true}]}`,
	} {
		if _, err := ParsePolicy([]byte(raw)); err != ErrPolicy {
			t.Fatal("invalid policy accepted")
		}
	}
}
