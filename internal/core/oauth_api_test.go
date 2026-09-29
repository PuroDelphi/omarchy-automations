package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"quatrro.local/automations/internal/local"
)

func TestGoogleConnectionAPIStatesAndRedaction(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	request := func(op, raw string) (any, error) {
		return e.Handle(ctx, local.Request{Op: op, Data: json.RawMessage(raw)})
	}
	input := `{"id":"account","backend":"file","client_id":"private-client-id","client_secret":"private-client-secret","refresh_token":"private-refresh-token"}`
	check := func(v any) {
		t.Helper()
		raw, _ := json.Marshal(v)
		for _, secret := range []string{"private-client", "private-refresh", "private-access"} {
			if strings.Contains(string(raw), secret) {
				t.Fatal("API leaked credential")
			}
		}
	}
	result, err := request("oauth.google.put", input)
	if err != nil {
		t.Fatal(err)
	}
	check(result)
	state := func(want string) {
		t.Helper()
		v, err := request("oauth.google.status", `{"id":"account"}`)
		if err != nil {
			t.Fatal(err)
		}
		check(v)
		if v.(map[string]any)["state"] != want {
			t.Fatalf("state: %v want %s", v, want)
		}
	}
	state("renewal_required")
	var grants int
	if err = e.db.QueryRow("SELECT count(*) FROM grants").Scan(&grants); err != nil || grants != 0 {
		t.Fatalf("provisioning grants: %d %v", grants, err)
	}
	now := time.Now()
	_, err = e.googleAccessTokenUsing(ctx, "account", func() time.Time { return now }, func(context.Context, string, string, string) (oauthToken, error) {
		return oauthToken{"private-access-token", "private-refresh-rotated", now.Add(time.Hour), ""}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	state("available")
	// Force renewal without waiting for expiry. Failure is persisted and survives
	// another call; invalid_grant must not cause repeated provider requests.
	calls := 0
	failed := func(context.Context, string, string, string) (oauthToken, error) {
		calls++
		return oauthToken{}, errOAuthReconnect
	}
	for i := 0; i < 2; i++ {
		_, err = e.googleAccessTokenUsing(ctx, "account", func() time.Time { return now.Add(time.Hour) }, failed)
		if err != errOAuthReconnect {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatal("terminal provider failure retried")
	}
	state("reconnect_required")
	reopened, err := Open(e.paths)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	v, err := reopened.googleConnectionStatus(ctx, []byte(`{"id":"account"}`))
	if err != nil || v.(map[string]any)["state"] != "reconnect_required" {
		t.Fatal("failure not persistent")
	}
	check(v)
	if _, err = request("oauth.google.put", input); err != nil {
		t.Fatal(err)
	}
	state("renewal_required")
	if _, err = request("secrets.delete", `{"id":"account"}`); err != nil {
		t.Fatal(err)
	}
	state("unavailable")
	_, err = request("oauth.google.put", `{"client_secret":{"private-secret":"value"}}`)
	if err != errOAuthCredentials {
		t.Fatal("malformed credential error not redacted")
	}
}

func TestGoogleConnectionTransientAndExpiredStates(t *testing.T) {
	for _, tc := range []struct {
		failure error
		state   string
	}{{errOAuthTemporary, "temporary_error"}, {errOAuthClient, "client_rejected"}, {errOAuthResponse, "invalid_response"}} {
		t.Run(tc.state, func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			putGoogleFixture(t, e, "refresh-token")
			_, err := e.googleAccessTokenUsing(ctx, "google-account", time.Now, func(context.Context, string, string, string) (oauthToken, error) { return oauthToken{}, tc.failure })
			if err != tc.failure {
				t.Fatal(err)
			}
			v, err := e.googleConnectionStatus(ctx, []byte(`{"id":"google-account"}`))
			if err != nil || v.(map[string]any)["state"] != tc.state {
				t.Fatal("failure state missing")
			}
			// Expired is derived from the stored timestamp, without network activity.
			c := googleCredential{Provider: "google", ClientID: "client", ClientSecret: "secret", RefreshToken: "refresh", AccessToken: "access", ExpiresAt: time.Now().Add(-time.Minute)}
			raw, _ := json.Marshal(c)
			req, _ := json.Marshal(secretWrite{ID: "google-account", Backend: "file", Value: string(raw)})
			if _, err = e.putSecret(ctx, req); err != nil {
				t.Fatal(err)
			}
			v, err = e.googleConnectionStatus(ctx, []byte(`{"id":"google-account"}`))
			if err != nil || v.(map[string]any)["state"] != "expired" {
				t.Fatal("expired state missing")
			}
		})
	}
}
