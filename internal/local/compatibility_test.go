package local

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientCompatibilityGateDoesNotSendOperation(t *testing.T) {
	for _, response := range []string{
		`{"version":1,"ok":false,"error":"incompatible protocol version"}`,
		`{"version":2,"contract":99,"ok":true,"data":{"handshake":"ready"}}`,
		`{"version":2,"contract":1,"ok":true,"data":{"handshake":"unknown"}}`,
	} {
		path := filepath.Join(t.TempDir(), "socket")
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		received := make(chan string, 1)
		go func() {
			conn, e := listener.Accept()
			if e != nil {
				received <- "accept failed"
				return
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(2 * time.Second))
			reader := bufio.NewReader(conn)
			hello, _ := reader.ReadString('\n')
			conn.Write([]byte(response + "\n"))
			rest, _ := io.ReadAll(reader)
			received <- hello + string(rest)
		}()
		_, err = Call(context.Background(), path, Request{Op: "emit", Data: json.RawMessage(`{"message":"private-payload"}`)})
		listener.Close()
		if !errors.Is(err, ErrIncompatible) {
			t.Fatal(err)
		}
		raw := <-received
		var hello Request
		if json.Unmarshal([]byte(raw), &hello) != nil || hello.Op != "hello" || len(hello.Data) != 0 || strings.Contains(raw, "private-payload") {
			t.Fatalf("operation leaked before compatibility: %q", raw)
		}
	}
}

func TestServerRejectsUnnegotiatedAndChangedContracts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "socket")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var effects atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, path, func(context.Context, Request) (any, error) { effects.Add(1); return true, nil })
	}()
	var conn net.Conn
	var err error
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.Dial("unix", path)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{"version":1,"op":"emit","data":{}}`,
		`{"version":2,"contract":1,"op":"emit","data":{}}`,
		`{"version":2,"contract":99,"op":"hello"}`,
		`{"version":2,"contract":1,"op":"hello"}` + "\n" + `{"version":2,"contract":99,"op":"emit"}`,
	} {
		conn, err = net.Dial("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		conn.SetDeadline(time.Now().Add(time.Second))
		conn.Write([]byte(raw + "\n"))
		reader := bufio.NewReader(conn)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(raw, "\n") {
			line, err = reader.ReadBytes('\n')
			if err != nil {
				t.Fatal(err)
			}
		}
		var response Response
		if json.Unmarshal(line, &response) != nil || response.OK || response.Error != ErrIncompatible.Error() {
			t.Fatalf("unexpected response %s", line)
		}
		conn.Close()
	}
	if effects.Load() != 0 {
		t.Fatal("incompatible request reached handler")
	}
	if _, err = Call(ctx, path, Request{Op: "emit"}); err != nil {
		t.Fatal(err)
	}
	if effects.Load() != 1 {
		t.Fatal("compatible request not dispatched exactly once")
	}
	cancel()
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}

func TestCancellationInterruptsHandshake(t *testing.T) {
	path := filepath.Join(t.TempDir(), "socket")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, e := listener.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		reader.ReadBytes('\n')
		cancel()
		io.Copy(io.Discard, reader)
	}()
	started := time.Now()
	_, err = Call(ctx, path, Request{Op: "emit"})
	if !errors.Is(err, context.Canceled) || time.Since(started) > time.Second {
		t.Fatal("handshake cancellation not bounded", err)
	}
	<-done
}
