package broker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

const SocketPath = "/run/quatrro-broker/control.sock"

var ErrUnavailable = errors.New("administrative broker unavailable or untrusted")

// Call has no socket override: production always talks to the root-owned system
// broker. No automatic retry is safe after the request may have reached it.
func Call(ctx context.Context, r Request) (Result, error) { return call(ctx, r, SocketPath, 0) }

func call(ctx context.Context, r Request, path string, serverUID uint32) (Result, error) {
	raw, err := json.Marshal(r)
	if err != nil {
		return Result{}, ErrRequest
	}
	if _, err = ParseRequest(raw); err != nil {
		return Result{}, ErrRequest
	}
	conn, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", path)
	if err != nil {
		return Result{}, ErrUnavailable
	}
	defer conn.Close()
	socket, ok := conn.(*net.UnixConn)
	if !ok {
		return Result{}, ErrUnavailable
	}
	descriptor, err := socket.SyscallConn()
	if err != nil {
		return Result{}, ErrUnavailable
	}
	var cred *unix.Ucred
	var peerErr error
	err = descriptor.Control(func(fd uintptr) { cred, peerErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED) })
	if err != nil || peerErr != nil || cred == nil || cred.Uid != serverUID {
		return Result{}, ErrUnavailable
	}
	deadline := time.Now().Add(25 * time.Second)
	if until, ok := ctx.Deadline(); ok && until.Before(deadline) {
		deadline = until
	}
	conn.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if ctx.Err() != nil {
		return Result{}, ErrUnavailable
	}
	if _, err = conn.Write(raw); err != nil {
		return Result{}, ErrOutcomeUnknown
	}
	if err = socket.CloseWrite(); err != nil {
		return Result{}, ErrOutcomeUnknown
	}
	response, err := io.ReadAll(io.LimitReader(conn, 8193))
	if err != nil || len(response) > 8192 {
		return Result{}, ErrOutcomeUnknown
	}
	return parseResponse(response, r)
}

func parseResponse(raw []byte, request Request) (Result, error) {
	var response Response
	if decode(raw, 8192, &response) != nil || response.Version != Version {
		return Result{}, ErrOutcomeUnknown
	}
	if !response.OK {
		if response.Result != nil {
			return Result{}, ErrOutcomeUnknown
		}
		switch response.Error {
		case "denied":
			return Result{}, ErrDenied
		case "invalid_request":
			return Result{}, ErrRequest
		case "status_unavailable":
			return Result{}, ErrStatus
		case "outcome_unknown":
			return Result{}, ErrOutcomeUnknown
		// An unknown failure cannot prove the operation was never dispatched.
		default:
			return Result{}, ErrOutcomeUnknown
		}
	}
	expectedState := "completed"
	if request.CheckOnly {
		expectedState = "authorized"
	}
	result := response.Result
	if response.Error != "" || result == nil || result.ID != request.ID || result.Unit != request.Unit || result.State != expectedState {
		return Result{}, ErrOutcomeUnknown
	}
	if request.Operation == "status" && !request.CheckOnly {
		if !statusValue.MatchString(result.LoadState) || !statusValue.MatchString(result.ActiveState) || !statusValue.MatchString(result.SubState) {
			return Result{}, ErrStatus
		}
	} else if result.LoadState != "" || result.ActiveState != "" || result.SubState != "" {
		return Result{}, ErrOutcomeUnknown
	}
	return *result, nil
}
