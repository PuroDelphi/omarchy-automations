package core

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

var errOAuthCancelled = errors.New("OAuth authorization cancelled or expired")
var errOAuthListener = errors.New("OAuth loopback listener unavailable")

type googleCallbackResult struct {
	code string
	err  error
}

type googleConsentListener struct {
	authorization *googleAuthorization
	url           string
	done          <-chan googleCallbackResult
	cancel        context.CancelFunc
}

// No provider calls or persistence occur here. The owner must exchange the code
// and save credentials only after successfully receiving a result.
func listenGoogleConsent(parent context.Context, clientID, clientSecret string, scopes []string) (*googleConsentListener, error) {
	if parent.Err() != nil {
		return nil, errOAuthCancelled
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, errOAuthListener
	}
	redirect := "http://" + listener.Addr().String() + "/oauth/callback"
	authorization, authURL, err := newGoogleAuthorization(clientID, clientSecret, redirect, scopes, time.Now())
	if err != nil {
		listener.Close()
		return nil, err
	}
	ctx, cancel := context.WithDeadline(parent, authorization.expires)
	callbacks := make(chan googleCallbackResult, 1)
	done := make(chan googleCallbackResult, 1)
	server := &http.Server{ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: time.Second, MaxHeaderBytes: 16384, ErrorLog: log.New(io.Discard, "", 0)}
	server.SetKeepAlivesEnabled(false)
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if ctx.Err() != nil || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "Invalid authorization response.")
			return
		}
		code, err := authorization.consumeCallback(r, time.Now())
		if err != nil && err != errOAuthConsentDenied {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "Invalid authorization response.")
			return
		}
		// Static text only: never reflect query parameters, codes or provider errors.
		io.WriteString(w, "Authorization response received. Return to Omarchy Automations.\nRespuesta recibida. Vuelve a Omarchy Automations.")
		callbacks <- googleCallbackResult{code: code, err: err}
	})
	failed := make(chan struct{}, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			failed <- struct{}{}
		}
	}()
	go func() {
		var result googleCallbackResult
		select {
		case result = <-callbacks:
		case <-ctx.Done():
			result.err = errOAuthCancelled
		case <-failed:
			result.err = errOAuthListener
		}
		if ctx.Err() != nil {
			result = googleCallbackResult{err: errOAuthCancelled}
		}
		cancel()
		closeCtx, closeCancel := context.WithTimeout(context.Background(), time.Second)
		if server.Shutdown(closeCtx) != nil {
			server.Close()
		}
		closeCancel()
		done <- result
		close(done)
	}()
	return &googleConsentListener{authorization: authorization, url: authURL, done: done, cancel: cancel}, nil
}
