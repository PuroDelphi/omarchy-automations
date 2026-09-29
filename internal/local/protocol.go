// Package local implements a bounded, same-UID Unix-socket control protocol.
package local

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const Version = 2

// Contract changes when operation semantics require matching clients and engine.
const Contract = 1

var ErrIncompatible = errors.New("incompatible Omarchy Automations components; update the engine, CLI and plugin together, restart quatrrod and reload the plugin")

const MaxMessage = 1 << 20

type Request struct {
	Version  int             `json:"version"`
	Contract int             `json:"contract"`
	Op       string          `json:"op"`
	Data     json.RawMessage `json:"data,omitempty"`
}
type Response struct {
	Version  int    `json:"version"`
	Contract int    `json:"contract"`
	OK       bool   `json:"ok"`
	Data     any    `json:"data,omitempty"`
	Error    string `json:"error,omitempty"`
}
type Handler func(context.Context, Request) (any, error)

func Serve(ctx context.Context, path string, handler Handler) error {
	// Hold an advisory lock before touching a stale socket; never unlink a live daemon.
	lock, err := os.OpenFile(filepath.Join(filepath.Dir(path), "daemon.lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("another daemon owns this runtime directory")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	if info, e := os.Lstat(path); e == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("refusing to replace non-socket")
		}
		if e = os.Remove(path); e != nil {
			return e
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		return err
	}
	defer listener.Close()
	if err = os.Chmod(path, 0600); err != nil {
		return err
	}
	go func() { <-ctx.Done(); listener.Close() }()
	slots := make(chan struct{}, 16)
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		select {
		case slots <- struct{}{}:
			go func() { defer func() { <-slots }(); serveConn(ctx, conn, handler) }()
		default:
			conn.Close()
		}
	}
}

func serveConn(ctx context.Context, conn *net.UnixConn, handler Handler) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	raw, err := conn.SyscallConn()
	if err != nil {
		return
	}
	var cred *syscall.Ucred
	var peerErr error
	if err = raw.Control(func(fd uintptr) {
		cred, peerErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil || peerErr != nil || cred == nil || cred.Uid != uint32(os.Getuid()) {
		return
	}
	reader := bufio.NewReaderSize(conn, MaxMessage+1)
	send := func(res Response) {
		res.Version = Version
		res.Contract = Contract
		out, err := json.Marshal(res)
		if err != nil || len(out) > MaxMessage {
			out = []byte(`{"version":2,"contract":1,"ok":false,"error":"response too large"}`)
		}
		conn.Write(append(out, '\n'))
	}
	first, err := readLine(reader)
	if err != nil {
		return
	}
	var hello Request
	if json.Unmarshal(first, &hello) != nil || hello.Version != Version || hello.Contract != Contract || hello.Op != "hello" || len(hello.Data) != 0 {
		send(Response{Error: ErrIncompatible.Error()})
		return
	}
	send(Response{OK: true, Data: map[string]string{"handshake": "ready"}})
	line, err := readLine(reader)
	if err != nil {
		return
	}
	var req Request
	res := Response{}
	if json.Unmarshal(line, &req) != nil {
		res.Error = "invalid request JSON"
	} else if req.Version != Version || req.Contract != Contract || req.Op == "hello" {
		res.Error = ErrIncompatible.Error()
	} else {
		requestCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		defer cancel()
		data, err := handler(requestCtx, req)
		if err != nil {
			res.Error = err.Error()
		} else {
			res.OK = true
			res.Data = data
		}
	}
	send(res)
}

func readLine(r io.Reader) ([]byte, error) {
	b, ok := r.(*bufio.Reader)
	if !ok {
		b = bufio.NewReaderSize(io.LimitReader(r, MaxMessage+1), MaxMessage+1)
	}
	line, err := b.ReadSlice('\n')
	if err != nil {
		return nil, err
	}
	if len(line) > MaxMessage {
		return nil, errors.New("message too large")
	}
	return line, nil
}

func Call(ctx context.Context, path string, req Request) (Response, error) {
	req.Version = Version
	req.Contract = Contract
	if req.Op == "hello" {
		return Response{}, errors.New("reserved operation")
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if err != nil {
		return Response{}, fmt.Errorf("backend unavailable: %w", err)
	}
	defer conn.Close()
	socket, ok := conn.(*net.UnixConn)
	if !ok {
		return Response{}, errors.New("untrusted backend connection")
	}
	descriptor, err := socket.SyscallConn()
	if err != nil {
		return Response{}, err
	}
	var cred *syscall.Ucred
	var peerErr error
	err = descriptor.Control(func(fd uintptr) {
		cred, peerErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil || peerErr != nil || cred == nil || cred.Uid != uint32(os.Getuid()) {
		return Response{}, errors.New("untrusted backend identity")
	}
	deadline := time.Now().Add(15 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	conn.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	hello, _ := json.Marshal(Request{Version: Version, Contract: Contract, Op: "hello"})
	if _, err = conn.Write(append(hello, '\n')); err != nil {
		return Response{}, err
	}
	reader := bufio.NewReaderSize(conn, MaxMessage+1)
	header, err := readLine(reader)
	if err != nil {
		if ctx.Err() != nil {
			return Response{}, ctx.Err()
		}
		return Response{}, errors.New("backend compatibility check unavailable; operation was not sent")
	}
	var ready Response
	if json.Unmarshal(header, &ready) != nil || ready.Version != Version || ready.Contract != Contract || !ready.OK || ready.Error != "" {
		return Response{}, ErrIncompatible
	}
	marker, ok := ready.Data.(map[string]any)
	if !ok || marker["handshake"] != "ready" {
		return Response{}, ErrIncompatible
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}
	if len(raw)+1 > MaxMessage {
		return Response{}, errors.New("message too large")
	}
	if _, err = conn.Write(append(raw, '\n')); err != nil {
		return Response{}, err
	}
	line, err := readLine(reader)
	if err != nil {
		return Response{}, err
	}
	var res Response
	if err = json.Unmarshal(line, &res); err != nil {
		return res, err
	}
	if res.Version != Version || res.Contract != Contract {
		return res, ErrIncompatible
	}
	if !res.OK {
		return res, errors.New(res.Error)
	}
	return res, nil
}
