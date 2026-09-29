package broker

import (
	"encoding/json"
	"strings"
)

// PreparedPolicy is an installation artifact, not an authorization grant.
// The installer must stop the broker while replacing policy and Polkit rules.
type PreparedPolicy struct {
	Policy      Policy `json:"policy"`
	PolkitRules string `json:"polkit_rules"`
}

func PreparePolicy(raw []byte) (PreparedPolicy, error) {
	p, err := ParsePolicy(raw)
	if err != nil {
		return PreparedPolicy{}, err
	}
	rules, err := RenderPolkitRules(p)
	if err != nil {
		return PreparedPolicy{}, err
	}
	// Use an explicit empty array in installation artifacts.
	if p.Rules == nil {
		p.Rules = []Rule{}
	}
	return PreparedPolicy{Policy: p, PolkitRules: string(rules)}, nil
}

// rejectDuplicateKeys prevents ambiguous administrative configuration and wire
// messages. The standard decoder otherwise silently keeps the last value.
func rejectDuplicateKeys(d *json.Decoder, depth int) error {
	if depth > 32 {
		return ErrRequest
	}
	token, err := d.Token()
	if err != nil {
		return ErrRequest
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return ErrRequest
			}
			name, ok := key.(string)
			if !ok || seen[strings.ToLower(name)] {
				return ErrRequest
			}
			seen[strings.ToLower(name)] = true
			if err = rejectDuplicateKeys(d, depth+1); err != nil {
				return err
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return ErrRequest
		}
	case '[':
		for d.More() {
			if err := rejectDuplicateKeys(d, depth+1); err != nil {
				return err
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return ErrRequest
		}
	default:
		return ErrRequest
	}
	return nil
}
