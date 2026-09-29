package broker

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBrokerTransportBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broker.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	var calls atomic.Int32
	go func() {
		done <- serve(ctx, listener, func(ctx context.Context, peer *PeerIdentity, r Request) (Result, error) {
			calls.Add(1)
			if peer.Peer().UID != uint32(os.Getuid()) {
				return Result{}, ErrDenied
			}
			if r.ID == "failure" {
				return Result{}, errors.New("sensitive internal detail")
			}
			return Result{ID: r.ID, Unit: r.Unit, State: "completed"}, nil
		})
	}()
	request := func(raw string) Response {
		t.Helper()
		conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err = conn.Write([]byte(raw)); err != nil {
			t.Fatal(err)
		}
		conn.CloseWrite()
		var result Response
		if err = json.NewDecoder(conn).Decode(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	valid := `{"version":1,"id":"job","unit":"backup.service","operation":"status"}`
	for _, raw := range []string{valid + valid, strings.Repeat("x", MaxRequest+1), `{"version":1,"id":"job","unit":"backup.service","operation":"status","uid":0}`} {
		r := request(raw)
		if r.OK || r.Error != "invalid_request" {
			t.Fatal("invalid request accepted", r)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid request reached executor")
	}
	result := request(valid)
	if !result.OK || result.Result == nil || result.Result.ID != "job" || calls.Load() != 1 {
		t.Fatal("valid request failed", result)
	}
	result = request(strings.Replace(valid, `"job"`, `"failure"`, 1))
	if result.OK || result.Error != "broker_unavailable" {
		t.Fatal("internal error leaked")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
}

func TestBrokerTransportCancellationClosesIncompleteRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broker.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, listener, func(context.Context, *PeerIdentity, Request) (Result, error) {
			return Result{}, errors.New("unexpected execution")
		})
	}()
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write([]byte(`{"version":`))
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("incomplete request held shutdown open")
	}
}
