package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDocumentedGoogleOAuth(t *testing.T) {
	raw, err := os.ReadFile("../../examples/use-cases/google-oauth.json")
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err = decode(raw, &c); err != nil {
		t.Fatal(err)
	}
	if err = c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Flows[0].Enabled {
		t.Fatal("example enabled")
	}
	c.Flows[0].Enabled = true
	e := testEngine(t)
	ctx := context.Background()
	raw, _ = json.Marshal(c)
	if _, err = e.saveDraft(ctx, raw); err != nil {
		t.Fatal(err)
	}
	var body string
	for _, calendar := range []string{"calendar-a@example.com", "calendar-b@example.com"} {
		raw, _ = json.Marshal(Event{Source: "local:calendar", Data: map[string]any{"start": "2030-01-07T09:00:00Z", "end": "2030-01-07T10:00:00Z", "calendar": calendar}})
		result, err := e.simulate(ctx, raw)
		if err != nil {
			t.Fatal("simulation required credentials", err)
		}
		report := result.(map[string]any)
		body = report["matches"].([]map[string]any)[0]["steps"].([]map[string]any)[0]["body"].(string)
		var rendered struct {
			Start string `json:"timeMin"`
			End   string `json:"timeMax"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		if err = json.Unmarshal([]byte(body), &rendered); err != nil || len(rendered.Items) != 1 || rendered.Items[0].ID != calendar || rendered.Start != "2030-01-07T09:00:00Z" || rendered.End != "2030-01-07T10:00:00Z" || report["effects_executed"] != false {
			t.Fatal(report, err)
		}
	}
	token := googleCredential{Provider: "google", ClientID: "fixture-client", ClientSecret: "fixture-client-secret", RefreshToken: "fixture-refresh", AccessToken: "fixture-access", ExpiresAt: time.Now().Add(time.Hour)}
	raw, _ = json.Marshal(token)
	request, _ := json.Marshal(secretWrite{ID: "google-calendar", Backend: "file", Value: string(raw)})
	if _, err = e.putSecret(ctx, request); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != c.Destinations[0].URL || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer fixture-access" {
			t.Fatal("wrong OAuth request")
		}
		payload, err := io.ReadAll(r.Body)
		if err != nil || string(payload) != body {
			t.Fatal("wrong request body", err)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"calendars":{}}`))}, nil
	})}
	result := e.sendWithClient(ctx, client, c.Destinations[0], "fixture-key", []byte(body))
	if result.Status != 200 || calls != 1 {
		t.Fatal(result, calls)
	}
	if _, err = e.deleteSecret(ctx, []byte(`{"id":"google-calendar"}`)); err != nil {
		t.Fatal(err)
	}
	result = e.sendWithClient(ctx, client, c.Destinations[0], "fixture-key", []byte(body))
	if result.Status != 0 || calls != 1 {
		t.Fatal("missing credential contacted provider")
	}
}
