package core

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestIngressStatusTracksListener(t *testing.T) {
	e := testEngine(t)
	if status := e.ingressStatus(); status.State != "disabled" || status.Address != "" || status.PublicExposure != "unverified" {
		t.Fatal(status)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- e.ServeIngress(ctx, "127.0.0.1:0") }()
	deadline := time.Now().Add(3 * time.Second)
	for e.ingressStatus().State != "listening" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := e.ingressStatus()
	if status.State != "listening" || status.Address == "127.0.0.1:0" || status.Address == "" {
		t.Fatal(status)
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + status.Address + "/not-a-hook")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatal(response.StatusCode)
	}
	value, err := e.status(context.Background())
	if err != nil || value.(map[string]any)["ingress"].(ingressStatus) != status {
		t.Fatal(value, err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("shutdown timeout")
	}
	if status = e.ingressStatus(); status.State != "stopped" || status.Address != "" || status.PublicExposure != "unverified" {
		t.Fatal(status)
	}
	if err := e.ServeIngress(context.Background(), "0.0.0.0:0"); err == nil {
		t.Fatal("non-loopback listener accepted")
	}
	if status = e.ingressStatus(); status.State != "failed" || status.Address != "" {
		t.Fatal(status)
	}
}
