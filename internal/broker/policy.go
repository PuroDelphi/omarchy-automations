// Package broker defines the optional privileged service boundary. Nothing in
// this package installs privileges or runs commands when parsing a policy.
package broker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const Version = 1
const MaxPolicy = 65536
const MaxRequest = 4096

var ErrPolicy = errors.New("invalid administrative broker policy")
var ErrDenied = errors.New("administrative operation denied")
var ErrRequest = errors.New("invalid administrative request")
var serviceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:@-]{0,230}\.service$`)
var requestID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type Rule struct {
	UID        uint32   `json:"uid"`
	Unit       string   `json:"unit"`
	Operations []string `json:"operations"`
}
type Policy struct {
	Version int    `json:"version"`
	Rules   []Rule `json:"rules"`
}
type Request struct {
	Version   int    `json:"version"`
	ID        string `json:"id"`
	Unit      string `json:"unit"`
	Operation string `json:"operation"`
	CheckOnly bool   `json:"check_only,omitempty"`
}

// Peer is supplied by trusted transport inspection, never by the JSON request.
type Peer struct {
	PID       int
	UID       uint32
	StartTime uint64
}

type Plan struct {
	ActionID   string
	PolkitArgs []string
	Executable string
	Args       []string
}

func decode(raw []byte, limit int, out any) error {
	if len(raw) == 0 || len(raw) > limit {
		return ErrRequest
	}
	if rejectDuplicateKeys(json.NewDecoder(bytes.NewReader(raw)), 0) != nil {
		return ErrRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil || decoder.Decode(new(any)) != io.EOF {
		return ErrRequest
	}
	return nil
}
func validUnit(unit string) bool {
	return serviceName.MatchString(unit) && !strings.Contains(unit, "@.service")
}
func validOperation(op string) bool {
	return op == "status" || op == "start" || op == "stop" || op == "restart"
}

func ParsePolicy(raw []byte) (Policy, error) {
	var p Policy
	if decode(raw, MaxPolicy, &p) != nil || p.Validate() != nil {
		return Policy{}, ErrPolicy
	}
	return p, nil
}
func (p Policy) Validate() error {
	if p.Version != Version || len(p.Rules) > 128 {
		return ErrPolicy
	}
	seen := map[string]bool{}
	for _, rule := range p.Rules {
		key := fmt.Sprintf("%d/%s", rule.UID, rule.Unit)
		if rule.UID == 0 || !validUnit(rule.Unit) || len(rule.Operations) == 0 || len(rule.Operations) > 4 || seen[key] {
			return ErrPolicy
		}
		seen[key] = true
		ops := map[string]bool{}
		for _, op := range rule.Operations {
			if !validOperation(op) || ops[op] {
				return ErrPolicy
			}
			ops[op] = true
		}
	}
	return nil
}
func ParseRequest(raw []byte) (Request, error) {
	var r Request
	if decode(raw, MaxRequest, &r) != nil || r.Version != Version || !requestID.MatchString(r.ID) || !validUnit(r.Unit) || !validOperation(r.Operation) {
		return Request{}, ErrRequest
	}
	return r, nil
}

// Authorize builds a plan after the local allowlist check. The caller still
// MUST perform Polkit authorization and revalidate the peer before execution.
func (p Policy) Authorize(peer Peer, r Request) (Plan, error) {
	if p.Validate() != nil || peer.UID == 0 || peer.PID <= 1 || peer.StartTime == 0 || r.Version != Version || !requestID.MatchString(r.ID) || !validUnit(r.Unit) || !validOperation(r.Operation) {
		return Plan{}, ErrDenied
	}
	permitted := false
	for _, rule := range p.Rules {
		if rule.UID == peer.UID && rule.Unit == r.Unit {
			for _, op := range rule.Operations {
				if op == r.Operation {
					permitted = true
				}
			}
		}
	}
	if !permitted {
		return Plan{}, ErrDenied
	}
	action := "org.quatrro.automations.service." + r.Operation
	polkit := []string{"--action-id", action, "--process", fmt.Sprintf("%d,%d,%d", peer.PID, peer.StartTime, peer.UID), "--detail", "unit", r.Unit}
	args := []string{"--system", "--no-ask-password", "--no-pager"}
	if r.Operation == "status" {
		args = append(args, "show", "--property=Id,LoadState,ActiveState,SubState", "--", r.Unit)
	} else {
		args = append(args, r.Operation, "--", r.Unit)
	}
	return Plan{ActionID: action, PolkitArgs: polkit, Executable: "/usr/bin/systemctl", Args: args}, nil
}
