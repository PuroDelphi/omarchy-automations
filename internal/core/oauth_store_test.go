package core

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func putGoogleFixture(t *testing.T, e *Engine, refresh string) {
	t.Helper()
	value, _ := json.Marshal(googleCredential{Provider: "google", ClientID: "client", ClientSecret: "secret", RefreshToken: refresh})
	req, _ := json.Marshal(secretWrite{ID: "google-account", Value: string(value), Backend: "file"})
	if _, err := e.putSecret(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}

func TestGoogleCredentialConcurrentRefreshAndReopen(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	putGoogleFixture(t, e, "initial-refresh")
	now := time.Now()
	clock := func() time.Time { return now }
	var calls atomic.Int32
	refresh := func(context.Context, string, string, string) (oauthToken, error) {
		calls.Add(1)
		return oauthToken{"new-access", "rotated-refresh", now.Add(time.Hour), ""}, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := e.googleAccessTokenUsing(ctx, "google-account", clock, refresh)
			if err != nil || token != "new-access" {
				t.Errorf("token retrieval failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("duplicate refreshes: %d", calls.Load())
	}
	raw, err := e.secret(ctx, "google-account")
	if err != nil {
		t.Fatal(err)
	}
	c, err := parseGoogleCredential(raw)
	if err != nil || c.RefreshToken != "rotated-refresh" {
		t.Fatal("rotation not persisted")
	}
	// A fresh Engine has no in-memory state from the first instance.
	reopened, err := Open(e.paths)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	token, err := reopened.googleAccessTokenUsing(ctx, "google-account", clock, refresh)
	if err != nil || token != "new-access" || calls.Load() != 1 {
		t.Fatal("persisted token not reused")
	}
	_, err = reopened.googleAccessTokenUsing(ctx, "google-account", func() time.Time { return now.Add(59*time.Minute + 30*time.Second) }, func(_ context.Context, _, _, r string) (oauthToken, error) {
		if r != "rotated-refresh" {
			t.Fatal("stale refresh token")
		}
		return oauthToken{}, errOAuthReconnect
	})
	if !errors.Is(err, errOAuthReconnect) {
		t.Fatal("near-expiry token incorrectly reused")
	}
}

func TestGoogleCredentialChangedDuringRefresh(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "replace", true: "delete"}[remove], func(t *testing.T) {
			e := testEngine(t)
			ctx := context.Background()
			putGoogleFixture(t, e, "initial-refresh")
			entered := make(chan struct{})
			resume := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				_, err := e.googleAccessTokenUsing(ctx, "google-account", time.Now, func(context.Context, string, string, string) (oauthToken, error) {
					close(entered)
					<-resume
					return oauthToken{"stale-access", "stale-refresh", time.Now().Add(time.Hour), ""}, nil
				})
				done <- err
			}()
			<-entered
			if remove {
				if _, err := e.deleteSecret(ctx, []byte(`{"id":"google-account"}`)); err != nil {
					t.Fatal(err)
				}
			} else {
				putGoogleFixture(t, e, "replacement-refresh")
			}
			close(resume)
			if err := <-done; !errors.Is(err, errOAuthChanged) {
				t.Fatalf("stale renewal accepted: %v", err)
			}
			raw, err := e.secret(ctx, "google-account")
			if remove {
				if err == nil {
					t.Fatal("deleted credentials resurrected")
				}
			} else {
				c, err := parseGoogleCredential(raw)
				if err != nil || c.RefreshToken != "replacement-refresh" || c.AccessToken != "" {
					t.Fatal("replacement overwritten")
				}
			}
		})
	}
}
