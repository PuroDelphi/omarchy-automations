package core

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"quatrro.local/automations/internal/broker"
)

func administrativeRequest(a Action, id string) (broker.Request, error) {
	if a.Kind != "system-service" || a.Executable != "" || len(a.Args) != 0 || a.Timeout != 0 || a.Script != "" || a.ScriptRevision != "" || len(a.ScriptValues) != 0 || len(a.ScriptBindings) != 0 || len(a.Directories) != 0 || a.WorkingDirectory != "" {
		return broker.Request{}, errors.New("administrative action accepts only a fixed service and operation")
	}
	request := broker.Request{Version: broker.Version, ID: id, Unit: a.Unit, Operation: a.Operation}
	raw, _ := json.Marshal(request)
	return broker.ParseRequest(raw)
}

func (e *Engine) performAdministrative(ctx context.Context, a Action, id string) (broker.Result, error) {
	request, err := administrativeRequest(a, id)
	if err != nil {
		return broker.Result{}, err
	}
	if e.brokerCaller != nil {
		return e.brokerCaller(ctx, request)
	}
	return broker.Call(ctx, request)
}

// checkAdministrative checks the current user's policy for one fixed operation.
// It neither grants a flow capability nor runs systemctl, even for status.
func (e *Engine) checkAdministrative(ctx context.Context, raw []byte) (any, error) {
	var input struct {
		Unit      string `json:"unit"`
		Operation string `json:"operation"`
	}
	if err := decode(raw, &input); err != nil {
		return nil, err
	}
	request, err := administrativeRequest(Action{Kind: "system-service", Unit: input.Unit, Operation: input.Operation}, "check-"+newID())
	if err != nil {
		return nil, err
	}
	request.CheckOnly = true
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	caller := e.brokerCaller
	if caller == nil {
		caller = broker.Call
	}
	result, err := caller(ctx, request)
	state := "unavailable"
	switch {
	case err == nil && result.State == "authorized" && result.Unit == request.Unit && result.ID == request.ID:
		state = "authorized"
	case errors.Is(err, broker.ErrDenied):
		state = "denied"
	case errors.Is(err, broker.ErrRequest):
		state = "unsupported"
	}
	return map[string]any{"state": state, "unit": request.Unit, "operation": request.Operation, "effects_executed": false}, nil
}
