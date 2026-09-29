package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"quatrro.local/automations/internal/core"
	"quatrro.local/automations/internal/local"
	"quatrro.local/automations/internal/paths"
	"quatrro.local/automations/internal/release"
	"quatrro.local/automations/internal/sandbox"
	"syscall"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		return json.NewEncoder(os.Stdout).Encode(release.Describe("quatrrod"))
	}
	if len(os.Args) == 2 && os.Args[1] == "--sandbox-runner" {
		syscall.Umask(0077)
		return sandbox.RunIO(os.Stdin, os.Stdout)
	}
	listen := flag.String("listen", "127.0.0.1:8791", "loopback webhook listener (empty disables HTTP)")
	flag.Parse()
	syscall.Umask(0077)
	p, err := paths.Resolve()
	if err != nil {
		return err
	}
	if err = p.Prepare(); err != nil {
		return err
	}
	lock, err := os.OpenFile(filepath.Join(p.State, "engine.lock"), os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("another motor owns this state directory")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	engine, err := core.Open(p)
	if err != nil {
		return err
	}
	defer engine.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	errors := make(chan error, 6)
	count := 5
	go func() { errors <- local.Serve(ctx, p.Socket(), engine.Handle) }()
	go func() { errors <- engine.Run(ctx) }()
	go func() { errors <- engine.RunMonitors(ctx) }()
	go func() { errors <- engine.RunTimers(ctx) }()
	go func() { errors <- engine.RunMaintenance(ctx) }()
	if *listen != "" {
		count++
		go func() { errors <- engine.ServeIngress(ctx, *listen) }()
	}
	err = <-errors
	cancel()
	for i := 1; i < count; i++ {
		<-errors
	}
	return err
}
