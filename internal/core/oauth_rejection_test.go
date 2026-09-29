package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGoogleRemoteRejectionDoesNotReplayOrInvalidateReplacement(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(map[bool]string{false: "reject", true: "replace-in-flight"}[replacement], func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			original := googleCredential{Provider: "google", ClientID: "client", ClientSecret: "client-secret", RefreshToken: "refresh-original", AccessToken: "access-original", ExpiresAt: time.Now().Add(time.Hour)}
			save := func(c googleCredential) {
				t.Helper()
				raw, _ := json.Marshal(c)
				req, _ := json.Marshal(secretWrite{ID: "account", Backend: "file", Value: string(raw)})
				if _, err := e.putSecret(ctx, req); err != nil {
					t.Fatal(err)
				}
			}
			save(original)
			calls := 0
			client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Authorization") != "Bearer access-original" {
					t.Fatal("wrong dispatched credential")
				}
				if replacement {
					updated := original
					updated.RefreshToken = "refresh-replacement"
					updated.AccessToken = "access-replacement"
					save(updated)
				}
				return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("private-provider-error")), Header: make(http.Header)}, nil
			})}
			d := Destination{URL: "https://www.googleapis.com/calendar/v3/test", Method: "POST", Auth: "oauth2-google", Secret: "account"}
			got := e.sendWithClient(ctx, client, d, "job", []byte(`{}`))
			if got.Status != 401 || got.Retry || calls != 1 || strings.Contains(got.Message, "private-provider") {
				t.Fatal("401 replayed or leaked", got)
			}
			v, err := e.googleConnectionStatus(ctx, []byte(`{"id":"account"}`))
			if err != nil {
				t.Fatal(err)
			}
			want := "token_rejected"
			if replacement {
				want = "available"
			}
			if v.(map[string]any)["state"] != want {
				t.Fatalf("state %v", v)
			}
			if !replacement {
				raw, err := e.secret(ctx, "account")
				if err != nil {
					t.Fatal(err)
				}
				c, err := parseGoogleCredential(raw)
				if err != nil || c.AccessToken != "" || c.RefreshToken != "refresh-original" {
					t.Fatal("rejection did not clear only access token")
				}
				refreshCalls := 0
				token, err := e.googleAccessTokenUsing(ctx, "account", time.Now, func(_ context.Context, _, _, r string) (oauthToken, error) {
					refreshCalls++
					if r != "refresh-original" {
						t.Fatal("lost refresh token")
					}
					return oauthToken{"access-new", r, time.Now().Add(time.Hour), ""}, nil
				})
				if err != nil || token != "access-new" || refreshCalls != 1 || calls != 1 {
					t.Fatal("subsequent renewal failed or delivery repeated")
				}
			}
		})
	}
}

func TestGoogleDispatchRevisionDetectsChangedCredentials(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	putGoogleFixture(t, e, "refresh")
	if _, err := e.googleDispatchRevision(ctx, "google-account", "stale-access"); err != errOAuthChanged {
		t.Fatal("stale credential accepted")
	}
}

func TestGoogleWorkerDeliveryAndTerminalRejection(t *testing.T) {
	for _, status := range []int{200, 401} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			cred := googleCredential{Provider: "google", ClientID: "client", ClientSecret: "secret", RefreshToken: "refresh", AccessToken: "worker-access", ExpiresAt: time.Now().Add(time.Hour)}
			raw, _ := json.Marshal(cred)
			req, _ := json.Marshal(secretWrite{ID: "account", Backend: "file", Value: string(raw)})
			if _, err := e.putSecret(ctx, req); err != nil {
				t.Fatal(err)
			}
			c := EmptyConfig()
			c.Destinations = []Destination{{ID: "google", URL: "https://www.googleapis.com/calendar/v3/test", Method: "POST", Auth: "oauth2-google", Secret: "account"}}
			c.Actions = []Action{{ID: "send", Kind: "http", Destination: "google", Body: `{"message":"{{data.message}}"}`}}
			c.Flows = []Flow{{ID: "flow", Source: "local:test", Enabled: true, Steps: []string{"send"}}}
			activateTest(t, e, c)
			calls := 0
			client := &http.Client{Transport: oauthTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), "worker-message") || r.Header.Get("Authorization") != "Bearer worker-access" {
					t.Fatal("worker payload/auth wrong")
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})}
			e.deliverySender = func(ctx context.Context, d Destination, key string, body []byte) deliveryResult {
				return e.sendWithClient(ctx, client, d, key, body)
			}
			if _, err := e.ingest(ctx, Event{Source: "local:test", Type: "test", Data: map[string]any{"message": "worker-message"}}); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 5; i++ {
				if err := e.tick(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var state string
			if err := e.db.QueryRow("SELECT state FROM executions").Scan(&state); err != nil {
				t.Fatal(err)
			}
			want := "completed"
			if status == 401 {
				want = "failed"
			}
			if state != want || calls != 1 {
				t.Fatalf("state=%s calls=%d", state, calls)
			}
		})
	}
}
