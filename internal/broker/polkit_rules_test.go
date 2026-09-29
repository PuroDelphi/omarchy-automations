package broker

import (
	"bytes"
	"os/exec"
	"testing"
)

func TestGeneratedPolkitRulesExactGrants(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node required for generated rule execution")
	}
	p := Policy{Version: 1, Rules: []Rule{{UID: 1000, Unit: "backup.service", Operations: []string{"status", "restart"}}}}
	rules, err := renderPolkitRules(p, func(uint32) (string, error) { return "test-user", nil })
	if err != nil {
		t.Fatal(err)
	}
	harness := `let handler; const polkit={Result:{YES:'yes',NO:'no'},addRule:f=>handler=f};
` + string(rules) + `
function check(user, id, unit, expected) {
 const got=handler({id,lookup:k=>k==='unit'?unit:undefined},{user});
 if (got!==expected) throw Error('rule boundary failed');
}
check('test-user','org.quatrro.automations.service.restart','backup.service','yes');
check('test-user','org.quatrro.automations.service.status','backup.service','yes');
check('other-user','org.quatrro.automations.service.restart','backup.service','no');
check('test-user','org.quatrro.automations.service.start','backup.service','no');
check('test-user','org.quatrro.automations.service.restart','other.service','no');
check('test-user','org.quatrro.automations.service.restart',undefined,'no');
check('test-user','org.quatrro.automations.service.restart.extra','backup.service','no');
check('test-user','org.freedesktop.systemd1.manage-units','backup.service',undefined);
`
	cmd := exec.Command(node, "-")
	cmd.Stdin = bytes.NewBufferString(harness)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated rule failed: %v %s", err, output)
	}
	empty, err := RenderPolkitRules(Policy{Version: 1})
	if err != nil || !bytes.Contains(empty, []byte("var grants = [];")) {
		t.Fatal("default policy grants access")
	}
	if _, err = renderPolkitRules(p, func(uint32) (string, error) { return "root", nil }); err != ErrPolicy {
		t.Fatal("root mapping accepted")
	}
}
