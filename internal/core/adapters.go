package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"quatrro.local/automations/internal/sandbox"

	"quatrro.local/automations/internal/adapters"
)

// Adapter embeds its manifest so the exported resource has a stable top-level ID.
// Only prepared local bytes are stored; source paths are not execution inputs.
type Adapter struct {
	adapters.Manifest
	Code     string `json:"code"`
	Revision string `json:"revision"`
}

func adapterRevision(a Adapter) string {
	a.Revision = ""
	return canonicalHash(a)
}

func (a Adapter) Validate() error {
	if err := a.Manifest.Validate(); err != nil {
		return err
	}
	// The same bounded UTF-8 code contract is used by local scripts and adapters.
	script := Script{ID: a.ID, Interpreter: a.Runtime, Code: a.Code, Parameters: []ScriptParameter{}}
	script.Revision = scriptRevision(script)
	if err := script.Validate(); err != nil {
		return err
	}
	if a.Revision != adapterRevision(a) {
		return errors.New("adapter revision does not match code and manifest")
	}
	return nil
}

func prepareAdapter(data []byte) (any, error) {
	var request struct {
		Path     string            `json:"path"`
		Manifest adapters.Manifest `json:"manifest"`
	}
	if err := decode(data, &request); err != nil {
		return nil, err
	}
	if err := request.Manifest.Validate(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(map[string]any{"path": request.Path, "id": request.Manifest.ID, "interpreter": request.Manifest.Runtime, "parameters": []ScriptParameter{}})
	if err != nil {
		return nil, err
	}
	prepared, err := prepareScript(raw)
	if err != nil {
		return nil, err
	}
	result := Adapter{Manifest: request.Manifest, Code: prepared.(Script).Code}
	result.Revision = adapterRevision(result)
	return result, result.Validate()
}

// finishAdapter stores a private event view with the step advance. It never
// rewrites the original inbox event shared by other flows.
func (e *Engine) finishAdapter(ctx context.Context, j job, data map[string]any) error {
	updated := j.Event
	updated.Data = make(map[string]any, len(j.Event.Data)+1)
	for key, value := range j.Event.Data {
		updated.Data[key] = value
	}
	updated.Data["adapter"] = data
	return e.finishJobContext(ctx, j, "pending", "adapter result persisted", &updated)
}

func selectedAdapter(c Config, a Action) (Adapter, error) {
	for _, adapter := range c.Adapters {
		if adapter.ID == a.Adapter && adapter.Revision == a.AdapterRevision {
			return adapter, adapter.Validate()
		}
	}
	return Adapter{}, errors.New("adapter action requires a registered exact revision")
}

type adapterOutput struct {
	data  bytes.Buffer
	limit int
}

func (b *adapterOutput) Write(data []byte) (int, error) {
	if b.data.Len()+len(data) > b.limit {
		return 0, errors.New("adapter output exceeds quota")
	}
	return b.data.Write(data)
}

func executeAdapter(ctx context.Context, adapter Adapter, event Event) (map[string]any, error) {
	if err := adapter.Validate(); err != nil {
		return nil, err
	}
	id := "run-" + newID()
	input, err := adapters.EncodeRequest(adapter.Manifest, id, adapters.Event{Source: event.Source, Type: event.Type, Data: event.Data})
	if err != nil {
		return nil, err
	}
	request := sandbox.Request{Version: 1, Executable: "/usr/bin/python3", Args: []string{"-I", "-S", "--", "/quatrro/script"}, Code: adapter.Code, Input: string(input), OutputLimit: adapter.OutputLimit}
	output := &adapterOutput{limit: adapter.OutputLimit}
	if err := runIsolatedTransport(ctx, Action{Timeout: adapter.TimeoutSeconds}, "quatrro-adapter-"+newID(), "", &request, output); err != nil {
		return nil, errors.New("isolated adapter failed, timed out or exceeded quota")
	}
	return adapters.DecodeResponse(adapter.Manifest, id, output.data.Bytes())
}
