package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ingress struct {
	engine *Engine
	mu     sync.Mutex
	window time.Time
	count  map[string]int
	slots  chan struct{}
}

func (e *Engine) IngressHandler() http.Handler {
	return &ingress{engine: e, window: time.Now(), count: map[string]int{}, slots: make(chan struct{}, 16)}
}
func (h *ingress) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	fail := func(status int, msg string) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": msg})
	}
	if r.Method != "POST" {
		fail(405, "POST required")
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/hooks/") {
		fail(404, "not found")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/hooks/")
	if !identifier.MatchString(id) {
		fail(404, "not found")
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	default:
		fail(503, "busy")
		return
	}
	h.mu.Lock()
	if time.Since(h.window) > time.Minute {
		h.window = time.Now()
		h.count = map[string]int{}
	}
	h.count[""]++
	limited := h.count[""] > 600
	h.mu.Unlock()
	if limited {
		w.Header().Set("Retry-After", "60")
		fail(429, "rate limit")
		return
	}
	c, err := h.engine.config(r.Context(), "active")
	if err != nil {
		fail(503, "configuration unavailable")
		return
	}
	var entry *Entry
	for i := range c.Entries {
		if c.Entries[i].ID == id && c.Entries[i].Enabled {
			entry = &c.Entries[i]
			break
		}
	}
	if entry == nil {
		fail(404, "not found")
		return
	}
	h.mu.Lock()
	h.count[id]++
	limited = h.count[id] > 120
	h.mu.Unlock()
	if limited {
		w.Header().Set("Retry-After", "60")
		fail(429, "rate limit")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 262144))
	if err != nil {
		fail(413, "body too large or unreadable")
		return
	}
	secret, err := h.engine.secret(r.Context(), entry.Secret)
	if err != nil {
		fail(503, "authentication unavailable")
		return
	}
	delivery, err := authenticate(*entry, r, raw, secret, time.Now())
	if err != nil {
		fail(401, "invalid authentication")
		return
	}
	data, err := parsePayload(entry.Format, r.Header.Get("Content-Type"), raw)
	if err != nil {
		fail(400, "invalid payload format")
		return
	}
	adapter, _ := inboundProvider(entry.Auth)
	if adapter.Challenge != nil {
		response, handled, err := adapter.Challenge(data)
		if err != nil {
			fail(400, "invalid provider challenge")
			return
		}
		if handled {
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	// Event headers are not covered by body-only provider signatures. Rules
	// must derive their event type from authenticated body content instead.
	eventType := "webhook"
	if value, ok := data["type"].(string); ok && value != "" {
		eventType = value
	}
	result, err := h.engine.ingest(r.Context(), Event{ID: delivery, Source: "entry:" + id, Type: eventType, Data: data})
	if err != nil {
		fail(503, "event could not be persisted")
		return
	}
	w.WriteHeader(adapter.SuccessStatus)
	if adapter.EmptyAcknowledgement {
		json.NewEncoder(w).Encode(map[string]any{})
		return
	}
	json.NewEncoder(w).Encode(result)
}
func parsePayload(format, contentType string, raw []byte) (map[string]any, error) {
	media, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	switch format {
	case "json":
		if media != "application/json" && !strings.HasSuffix(media, "+json") {
			return nil, errors.New("JSON content type required")
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil || data == nil {
			return nil, errors.New("JSON object required")
		}
		return data, nil
	case "form":
		if media != "application/x-www-form-urlencoded" {
			return nil, errors.New("form content type required")
		}
		values, err := url.ParseQuery(string(raw))
		if err != nil {
			return nil, err
		}
		data := map[string]any{}
		if len(values) > 128 {
			return nil, errors.New("too many fields")
		}
		for k, v := range values {
			if len(v) != 1 {
				return nil, errors.New("duplicate form field")
			}
			data[k] = v[0]
		}
		return data, nil
	case "xml":
		if media != "application/xml" && media != "text/xml" && !strings.HasSuffix(media, "+xml") {
			return nil, errors.New("XML content type required")
		}
		return parseXML(raw)
	case "multipart":
		if media != "multipart/form-data" {
			return nil, errors.New("multipart content type required")
		}
		return parseMultipart(raw, params["boundary"])
	case "raw":
		return map[string]any{"body": string(raw), "content_type": media}, nil
	}
	return nil, errors.New("unsupported format")
}

type ingressStatus struct {
	State          string `json:"state"`
	Address        string `json:"address,omitempty"`
	PublicExposure string `json:"public_exposure"`
}

func (e *Engine) ingressStatus() ingressStatus {
	if value := e.ingressState.Load(); value != nil {
		return value.(ingressStatus)
	}
	return ingressStatus{State: "disabled", PublicExposure: "unverified"}
}

func (e *Engine) ServeIngress(ctx context.Context, address string) (err error) {
	e.ingressState.Store(ingressStatus{State: "starting", PublicExposure: "unverified"})
	defer func() {
		state := "stopped"
		if err != nil {
			state = "failed"
		}
		e.ingressState.Store(ingressStatus{State: state, PublicExposure: "unverified"})
	}()
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("ingress must bind a loopback IP; expose through an authenticated TLS proxy")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	e.ingressState.Store(ingressStatus{State: "listening", Address: listener.Addr().String(), PublicExposure: "unverified"})
	server := &http.Server{Handler: e.IngressHandler(), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 20 * time.Second, MaxHeaderBytes: 16384}
	go func() {
		<-ctx.Done()
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(stop)
	}()
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
