package core

import "quatrro.local/automations/internal/sandbox"

func prepareDirectory(data []byte) (any, error) {
	var request struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Access string `json:"access"`
	}
	if err := decode(data, &request); err != nil {
		return nil, err
	}
	return sandbox.PrepareDirectory(request.Source, request.Target, request.Access)
}
