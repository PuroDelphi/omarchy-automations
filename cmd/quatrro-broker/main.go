// quatrro-broker is optional and runs only through a root-owned systemd socket.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"quatrro.local/automations/internal/broker"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) == 2 && os.Args[1] == "--prepare-policy" {
		return preparePolicy(os.Stdin, os.Stdout)
	}

	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("quatrro-broker protocol", broker.Version)
		return nil
	}
	if len(os.Args) != 1 {
		return errors.New("broker accepts only systemd socket activation")
	}
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		return errors.New("broker requires system service identity")
	}
	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil || pid != os.Getpid() || os.Getenv("LISTEN_FDS") != "1" {
		return errors.New("broker requires one systemd activation socket")
	}
	os.Unsetenv("LISTEN_PID")
	os.Unsetenv("LISTEN_FDS")
	os.Unsetenv("LISTEN_FDNAMES")
	file := os.NewFile(3, "broker-listener")
	listener, err := net.FileListener(file)
	file.Close()
	if err != nil {
		return errors.New("invalid broker activation socket")
	}
	defer listener.Close()
	unixListener, ok := listener.(*net.UnixListener)
	if !ok || unixListener.Addr().String() != broker.SocketPath {
		return errors.New("unexpected broker activation address")
	}
	unixListener.SetUnlinkOnClose(false)
	if _, err = broker.LoadPolicy(broker.DefaultPolicyPath); err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	return broker.Serve(ctx, unixListener)
}

// Read-only preparation is available without privilege or a system bus.
func preparePolicy(input io.Reader, output io.Writer) error {
	raw, err := io.ReadAll(io.LimitReader(input, broker.MaxPolicy+1))
	if err != nil {
		return broker.ErrPolicy
	}
	prepared, err := broker.PreparePolicy(raw)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(prepared)
}
