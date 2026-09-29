package core

import (
	"context"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"path/filepath"
	"quatrro.local/automations/internal/broker"
	"quatrro.local/automations/internal/local"
	"quatrro.local/automations/internal/paths"
	"quatrro.local/automations/internal/release"
	"sync"
	"sync/atomic"
	"time"
)

const Version = release.Version

type Engine struct {
	brokerCaller   func(context.Context, broker.Request) (broker.Result, error)
	db             *sql.DB
	paths          paths.Paths
	started        time.Time
	mutation       sync.Mutex
	oauthMu        sync.Mutex
	consentMu      sync.Mutex
	consent        *googleConsentSession
	consentClosed  bool
	runningMu      sync.Mutex
	running        map[string]context.CancelFunc
	lastError      atomic.Value
	ingressState   atomic.Value
	actionRunner   func(context.Context, Action, Event) error
	deliverySender func(context.Context, Destination, string, []byte) deliveryResult
	cpu            map[string]cpuSample
}

func Open(p paths.Paths) (*Engine, error) {
	db, err := sql.Open("sqlite", filepath.Join(p.State, "quatrro.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	err = migrate(context.Background(), db)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Engine{db: db, paths: p, started: time.Now(), running: map[string]context.CancelFunc{}, cpu: map[string]cpuSample{}}, nil
}
func (e *Engine) Close() error { e.closeGoogleConsent(); return e.db.Close() }
func (e *Engine) Handle(ctx context.Context, r local.Request) (any, error) {
	switch r.Op {
	case "administration.check":
		return e.checkAdministrative(ctx, r.Data)
	case "oauth.google.current":
		return e.currentGoogleConsent(r.Data)
	case "oauth.google.begin":
		return e.startGoogleConsent(ctx, r.Data, nil)
	case "oauth.google.session":
		return e.googleConsentStatus(r.Data, false)
	case "oauth.google.cancel":
		return e.googleConsentStatus(r.Data, true)
	case "oauth.google.put":
		return e.putGoogleConnection(ctx, r.Data)
	case "oauth.google.status":
		return e.googleConnectionStatus(ctx, r.Data)
	case "adapters.prepare":
		return prepareAdapter(r.Data)
	case "directories.prepare":
		return prepareDirectory(r.Data)
	case "scripts.prepare":
		return prepareScript(r.Data)
	case "preferences.get":
		return e.preferences(ctx)
	case "preferences.set":
		return e.setPreferences(ctx, r.Data)
	case "status":
		return e.status(ctx)
	case "config.get":
		return e.config(ctx, "draft")
	case "config.active":
		return e.config(ctx, "active")
	case "config.save":
		return e.saveDraft(ctx, r.Data)
	case "config.preview":
		return e.preview(ctx)
	case "config.activate":
		return e.activate(ctx, r.Data)
	case "permissions.revoke":
		return e.revoke(ctx, r.Data)
	case "secrets.put":
		return e.putSecret(ctx, r.Data)
	case "secrets.list":
		return e.listSecrets(ctx)
	case "secrets.delete":
		return e.deleteSecret(ctx, r.Data)
	case "emit":
		return e.emit(ctx, r.Data)
	case "simulate":
		return e.simulate(ctx, r.Data)
	case "control":
		return e.control(ctx, r.Data)
	case "cancel":
		return e.cancelJobs(ctx)
	case "history":
		return e.history(ctx)
	case "timers.status":
		return e.timerStatus(ctx)
	case "monitors.reset":
		return e.resetMonitor(ctx, r.Data)
	case "monitors.status":
		return e.monitorStatus(ctx)
	case "permissions.list":
		return e.permissions(ctx)
	case "history.detail":
		return e.detail(ctx, r.Data)
	case "delivery.retry":
		return e.retry(ctx, r.Data)
	case "config.import":
		return e.importConfig(ctx, r.Data)
	case "config.import-file":
		return e.configFile(ctx, r.Data, false)
	case "config.export-file":
		return e.configFile(ctx, r.Data, true)
	case "diagnostics":
		return e.diagnostics(ctx)
	case "diagnostics.export-file":
		return e.diagnosticsFile(ctx, r.Data)
	case "queue.inspect":
		return e.queueInspect(ctx, r.Data)
	case "storage.status":
		return e.storageStatus(ctx)
	case "storage.policy":
		return e.setStoragePolicy(ctx, r.Data)
	default:
		return nil, fmt.Errorf("unknown operation: %s", r.Op)
	}
}
