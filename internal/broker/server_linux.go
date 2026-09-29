package broker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type Response struct {
	Version int     `json:"version"`
	OK      bool    `json:"ok"`
	Result  *Result `json:"result,omitempty"`
	Error   string  `json:"error,omitempty"`
}

type operationHandler func(context.Context, *PeerIdentity, Request) (Result, error)

// Serve uses an already bound Unix listener supplied by socket activation. It
// never creates, chmods or replaces a path provided by an unprivileged request.
func Serve(ctx context.Context, listener *net.UnixListener) error {
	return serve(ctx, listener, Execute)
}

func serve(parent context.Context, listener *net.UnixListener, execute operationHandler) error {
	ctx, cancel := context.WithCancel(parent)
	var mu sync.Mutex
	connections := map[*net.UnixConn]bool{}
	var jobs sync.WaitGroup
	slots := make(chan struct{}, 8)
	stop := func() {
		listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for conn := range connections {
			conn.Close()
		}
	}
	defer func() { cancel(); stop(); jobs.Wait() }()
	go func() { <-ctx.Done(); stop() }()
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return errors.New("broker listener unavailable")
		}
		select {
		case slots <- struct{}{}:
			mu.Lock()
			connections[conn] = true
			mu.Unlock()
			jobs.Add(1)
			go func() {
				defer jobs.Done()
				defer func() { <-slots; mu.Lock(); delete(connections, conn); mu.Unlock() }()
				serveConnection(ctx, conn, execute)
			}()
		default:
			conn.Close()
		}
	}
}

func serveConnection(parent context.Context, conn *net.UnixConn, execute operationHandler) {
	defer conn.Close()
	ctx, cancel := context.WithTimeout(parent, 22*time.Second)
	defer cancel()
	reply := func(result *Result, code string) {
		conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		json.NewEncoder(conn).Encode(Response{Version: Version, OK: code == "", Result: result, Error: code})
	}
	identity, err := IdentifyPeer(conn)
	if err != nil {
		reply(nil, "denied")
		return
	}
	defer identity.Close()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	// The client must half-close its write side. EOF frames exactly one JSON
	// request, so concatenated requests are rejected before any effect.
	raw, err := io.ReadAll(io.LimitReader(conn, MaxRequest+1))
	if err != nil || len(raw) > MaxRequest {
		reply(nil, "invalid_request")
		return
	}
	request, err := ParseRequest(raw)
	if err != nil {
		reply(nil, "invalid_request")
		return
	}
	if ctx.Err() != nil || identity.Revalidate() != nil {
		reply(nil, "denied")
		return
	}
	result, err := execute(ctx, identity, request)
	if err != nil {
		code := "broker_unavailable"
		switch {
		case errors.Is(err, ErrDenied), errors.Is(err, ErrPolicy):
			code = "denied"
		case errors.Is(err, ErrRequest):
			code = "invalid_request"
		case errors.Is(err, ErrOutcomeUnknown):
			code = "outcome_unknown"
		case errors.Is(err, ErrStatus):
			code = "status_unavailable"
		}
		reply(nil, code)
		return
	}
	reply(&result, "")
}
