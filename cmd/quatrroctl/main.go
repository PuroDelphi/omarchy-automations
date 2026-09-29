package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"quatrro.local/automations/internal/hooks"
	"quatrro.local/automations/internal/local"
	"quatrro.local/automations/internal/paths"
	"quatrro.local/automations/internal/release"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: quatrroctl <status|OP> [--stdin|JSON]; protocol %d", local.Version)
	}
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		return json.NewEncoder(os.Stdout).Encode(release.Describe("quatrroctl"))
	}
	var err error
	if os.Args[1] == "hook" {
		if len(os.Args) < 3 {
			return fmt.Errorf("usage: quatrroctl hook <name> [arguments...]")
		}
		p, err := paths.Resolve()
		if err != nil {
			return err
		}
		return hooks.Send(context.Background(), p.Socket(), os.Args[2], os.Args[3:])
	}
	var data []byte
	if len(os.Args) > 2 {
		if os.Args[2] == "--stdin" {
			data, err = io.ReadAll(io.LimitReader(os.Stdin, local.MaxMessage+1))
			if err != nil {
				return err
			}
		} else {
			data = []byte(os.Args[2])
		}
		if !json.Valid(data) {
			return fmt.Errorf("payload must be valid JSON")
		}
	}
	operation := os.Args[1]
	if operation == "ui" {
		if len(os.Args) != 3 || os.Args[2] != "--stdin" {
			return local.ErrIncompatible
		}
		operation, data, err = decodeUI(data)
		if err != nil {
			return err
		}
	}
	p, err := paths.Resolve()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := local.Call(ctx, p.Socket(), local.Request{Op: operation, Data: data})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(res.Data)
}
