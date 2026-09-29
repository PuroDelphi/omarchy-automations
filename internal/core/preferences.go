package core

import (
	"context"
	"database/sql"
	"errors"
)

func (e *Engine) preferences(ctx context.Context) (any, error) {
	language := "en"
	var stored string
	err := e.db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key='ui_language'").Scan(&stored)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if stored == "es" {
		language = "es"
	}
	return map[string]string{"language": language}, nil
}

func (e *Engine) setPreferences(ctx context.Context, raw []byte) (any, error) {
	var request struct {
		Language string `json:"language"`
	}
	if err := decode(raw, &request); err != nil {
		return nil, err
	}
	if request.Language != "en" && request.Language != "es" {
		return nil, errors.New("language must be en or es")
	}
	e.mutation.Lock()
	defer e.mutation.Unlock()
	_, err := e.db.ExecContext(ctx, "INSERT INTO settings(key,value) VALUES('ui_language',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", request.Language)
	if err != nil {
		return nil, err
	}
	return map[string]string{"language": request.Language}, nil
}
