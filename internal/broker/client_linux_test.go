package broker

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBrokerClientProtocolAndAmbiguousResults(t *testing.T) {
	request := Request{Version: 1, ID: "job", Unit: "backup.service", Operation: "restart"}
	cases := []struct {
		raw  string
		want error
	}{
		{`{"version":1,"ok":true,"result":{"id":"job","unit":"backup.service","state":"completed"}}`, nil},
		{`{"version":1,"ok":false,"error":"denied"}`, ErrDenied},
		{`{"version":1,"ok":false,"error":"sensitive diagnostic"}`, ErrOutcomeUnknown},
		{`{"version":2,"ok":true}`, ErrOutcomeUnknown},
		{`{"version":1,"ok":true,"result":{"id":"other","unit":"backup.service","state":"completed"}}`, ErrOutcomeUnknown},
		{`{"version":1,"ok":true,"result":{"id":"job","unit":"other.service","state":"completed"}}`, ErrOutcomeUnknown},
		{`{"version":1,"ok":false,"error":"denied","result":{"id":"job"}}`, ErrOutcomeUnknown},
		{`{} {}`, ErrOutcomeUnknown},
		{strings.Repeat("x", 8193), ErrOutcomeUnknown},
		{``, ErrOutcomeUnknown},
	}
	for _, tc := range cases {
		path := filepath.Join(t.TempDir(), "broker.sock")
		listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer listener.Close()
			conn, err := listener.AcceptUnix()
			if err != nil {
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(3 * time.Second))
			raw, _ := io.ReadAll(conn)
			var got Request
			if json.Unmarshal(raw, &got) != nil || got != request {
				return
			}
			conn.Write([]byte(tc.raw))
		}()
		_, err = call(context.Background(), request, path, uint32(os.Getuid()))
		<-done
		if err != tc.want {
			t.Fatalf("unexpected result: got %v want %v", err, tc.want)
		}
	}
}

func TestBrokerClientRejectsUntrustedServerBeforeSend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broker.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan int, 1)
	go func() {
		conn, err := listener.AcceptUnix()
		if err != nil {
			received <- -1
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		raw, _ := io.ReadAll(conn)
		received <- len(raw)
	}()
	_, err = call(context.Background(), Request{Version: 1, ID: "job", Unit: "backup.service", Operation: "restart"}, path, uint32(os.Getuid()+1))
	if err != ErrUnavailable || <-received != 0 {
		t.Fatal("request sent to untrusted server")
	}
}

func TestBrokerClientCancellationAfterSendIsUncertain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broker.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.AcceptUnix()
		if err != nil {
			return
		}
		defer conn.Close()
		io.ReadAll(conn)
		cancel()
	}()
	if _, err = call(ctx, Request{Version: 1, ID: "job", Unit: "backup.service", Operation: "restart"}, path, uint32(os.Getuid())); err != ErrOutcomeUnknown {
		t.Fatal("cancelled dispatched operation not uncertain", err)
	}
	<-done
}
