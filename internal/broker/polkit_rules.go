package broker

import (
	"encoding/json"
	"os/user"
	"strconv"
	"strings"
)

// RenderPolkitRules resolves account names locally for Polkit's subject.user.
// The broker independently checks the kernel UID against the root-owned policy.
// Installation must update policy and generated rules together while stopped.
func RenderPolkitRules(p Policy) ([]byte, error) {
	return renderPolkitRules(p, func(uid uint32) (string, error) {
		u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
		if err != nil {
			return "", err
		}
		return u.Username, nil
	})
}

func renderPolkitRules(p Policy, lookup func(uint32) (string, error)) ([]byte, error) {
	if p.Validate() != nil {
		return nil, ErrPolicy
	}
	type grant struct {
		User    string   `json:"user"`
		Unit    string   `json:"unit"`
		Actions []string `json:"actions"`
	}
	grants := []grant{}
	for _, rule := range p.Rules {
		name, err := lookup(rule.UID)
		if err != nil || name == "" || name == "root" || len(name) > 256 || strings.ContainsAny(name, "\x00\r\n") {
			return nil, ErrPolicy
		}
		actions := []string{}
		for _, op := range rule.Operations {
			actions = append(actions, "org.quatrro.automations.service."+op)
		}
		grants = append(grants, grant{User: name, Unit: rule.Unit, Actions: actions})
	}
	raw, err := json.Marshal(grants)
	if err != nil {
		return nil, ErrPolicy
	}
	// Data is encoded as JSON; account names can never inject rule source.
	source := `// Generated from the Quatrro broker allowlist. No interactive grants.
polkit.addRule(function(action, subject) {
    if (action.id.indexOf("org.quatrro.automations.service.") !== 0) return;
    var grants = ` + string(raw) + `;
    for (var i = 0; i < grants.length; i++) {
        var grant = grants[i];
        if (subject.user === grant.user && action.lookup("unit") === grant.unit && grant.actions.indexOf(action.id) !== -1) return polkit.Result.YES;
    }
    return polkit.Result.NO;
});
`
	return []byte(source), nil
}
