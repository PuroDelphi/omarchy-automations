package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileMetricsAndNoSymlinkTraversal(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.bin")
	now := time.Now()
	m := Monitor{Metric: "file_exists", Path: path}
	if value, err := extendedMetric(ctx, m, now); err != nil || value != 0 {
		t.Fatal(value, err)
	}
	if err := os.WriteFile(path, []byte("fixture-file-data"), 0600); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(path, now.Add(-time.Hour), now.Add(-time.Hour))
	for metric, want := range map[string]float64{"file_exists": 100, "file_age": 3600, "file_size": 17} {
		m.Metric = metric
		value, err := extendedMetric(ctx, m, now)
		if err != nil || value != want {
			t.Fatal(metric, value, err)
		}
	}
	link := filepath.Join(dir, "link")
	os.Symlink(path, link)
	m.Path = link
	if _, err := extendedMetric(ctx, m, now); err == nil {
		t.Fatal("followed final symlink")
	}
	linkDir := filepath.Join(dir, "linked-dir")
	os.Symlink(dir, linkDir)
	m.Path = filepath.Join(linkDir, "backup.bin")
	if _, err := extendedMetric(ctx, m, now); err == nil {
		t.Fatal("followed parent symlink")
	}
	m.Path = dir
	if _, err := extendedMetric(ctx, m, now); err == nil {
		t.Fatal("accepted directory as file")
	}
	m.Path = path
	m.Metric = "file_age"
	os.Chtimes(path, now.Add(time.Hour), now.Add(time.Hour))
	if _, err := extendedMetric(ctx, m, now); err == nil {
		t.Fatal("future timestamp treated as normal")
	}
}

func TestTemperatureAndProcessMetrics(t *testing.T) {
	for raw, want := range map[string]float64{"42000\n": 42, "-1000": -1, "0": 0} {
		value, err := parseTemperature([]byte(raw))
		if err != nil || value != want {
			t.Fatal(value, err)
		}
	}
	for _, raw := range []string{"NaN", "+Inf", "1000001", "-273151", "unavailable"} {
		if _, err := parseTemperature([]byte(raw)); err == nil {
			t.Fatal("bad sensor accepted")
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	value, err := processPresence(context.Background(), executable)
	if err != nil || value != 100 {
		t.Fatal("current user process missing", value, err)
	}
	value, err = processPresence(context.Background(), "/nonexistent/quatrro-fixture")
	if value != 0 {
		t.Fatal(value, err)
	}
}

func TestConnectivityUsesGrantedDestinationAndCredentials(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	secret := "connectivity-test-secret"
	raw, _ := json.Marshal(map[string]string{"id": "health", "backend": "file", "value": secret})
	if _, err := e.putSecret(ctx, raw); err != nil {
		t.Fatal(err)
	}
	calls := 0
	status := 204
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "HEAD" || r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("unexpected health request", r.Method)
		}
		w.WriteHeader(status)
	}))
	defer server.Close()
	address := strings.TrimPrefix(server.URL, "https://")
	c := EmptyConfig()
	c.Destinations = []Destination{{ID: "health", URL: server.URL, Method: "POST", Auth: "bearer", Secret: "health", PrivateHosts: []string{address}}}
	m := Monitor{ID: "health", Metric: "connectivity", Destination: "health", Threshold: 50, Recovery: 75, Interval: 5, Cooldown: 5, Enabled: true}
	c.Monitors = []Monitor{m}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	client := outboundClient(c.Destinations[0], net.DefaultResolver)
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: server.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs, MinVersion: tls.VersionTLS12}
	value, err := e.connectivityMetric(ctx, c, m, client)
	if err != nil || value != 100 || calls != 1 {
		t.Fatal(value, calls, err)
	}
	status = 503
	value, err = e.connectivityMetric(ctx, c, m, client)
	if err != nil || value != 0 {
		t.Fatal(value, err)
	}
	before := monitorHash(c, m)
	c.Destinations[0].URL += "/changed"
	if monitorHash(c, m) == before {
		t.Fatal("destination change reused monitor permission")
	}
	c.Destinations[0].PrivateHosts = nil
	blocked := outboundClient(c.Destinations[0], net.DefaultResolver)
	if _, err = e.connectivityMetric(ctx, c, m, blocked); err == nil {
		t.Fatal("policy rejection did not report unavailable")
	}
}

func TestJournalMetadataAtomicityRevocationAndReset(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	m := Monitor{ID: "logs", Metric: "journal", Unit: "fixture.service", Priority: 3, Interval: 5, Enabled: true}
	c.Monitors = []Monitor{m}
	c.Flows[0].Source = "monitor:logs"
	activateTest(t, e, c)
	raw := []byte(`{"__CURSOR":"cursor-1","__REALTIME_TIMESTAMP":"1800000000000000","PRIORITY":"3","_SYSTEMD_USER_UNIT":"fixture.service","MESSAGE":"private-message-never-copy"}`)
	records, err := parseJournal(raw)
	if err != nil {
		t.Fatal(err)
	}
	state := journalState{Hash: monitorHash(c, m), Since: 1800000000000000, Next: 1800000005}
	if _, err = e.db.Exec(`CREATE TRIGGER fail_journal BEFORE INSERT ON journal_state BEGIN SELECT RAISE(ABORT,'fixture'); END`); err != nil {
		t.Fatal(err)
	}
	if err = e.commitJournal(ctx, m, state, records); err == nil {
		t.Fatal("expected rollback")
	}
	var count int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 0 {
		t.Fatal("journal event escaped transaction")
	}
	e.db.Exec("DROP TRIGGER fail_journal")
	if err = e.commitJournal(ctx, m, state, records); err != nil {
		t.Fatal(err)
	}
	var payload string
	e.db.QueryRow("SELECT payload FROM events").Scan(&payload)
	if strings.Contains(payload, "private-message") {
		t.Fatal("journal message leaked")
	}
	state.PreviousHash = state.Hash
	if err = e.commitJournal(ctx, m, state, records); err != nil {
		t.Fatal(err)
	}
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 1 {
		t.Fatal("journal replay not deduplicated")
	}
	if _, err = e.resetMonitor(ctx, []byte(`{"id":"logs"}`)); err != nil {
		t.Fatal(err)
	}
	records[0].Cursor = "cursor-2"
	if err = e.commitJournal(ctx, m, state, records); err != nil {
		t.Fatal(err)
	}
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 1 {
		t.Fatal("in-flight sample survived reset")
	}
	e.revoke(ctx, []byte(`{"scope":"monitor:logs"}`))
	state.PreviousHash = ""
	if err = e.commitJournal(ctx, m, state, records); err != nil {
		t.Fatal(err)
	}
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 1 {
		t.Fatal("revoked journal emitted")
	}
	buffer := &boundedBuffer{Limit: 8}
	if _, err := io.Copy(buffer, strings.NewReader("nine-bytes")); err == nil || len(buffer.Bytes()) > 8 {
		t.Fatal("output quota bypassed")
	}
}

func TestHostJournalSelectedUnit(t *testing.T) {
	hostIsolation(t)
	e := testEngine(t)
	ctx := context.Background()
	unit := "quatrro-journal-test-" + newID() + ".service"
	c := notificationConfig()
	m := Monitor{ID: "logs", Metric: "journal", Unit: unit, Priority: 3, Interval: 5, Enabled: true}
	c.Monitors = []Monitor{m}
	c.Flows[0].Source = "monitor:logs"
	activateTest(t, e, c)
	start := time.Now()
	if err := e.sampleMonitors(ctx, start); err != nil {
		t.Fatal(err)
	}
	if err := runBounded(ctx, 5*time.Second, "systemd-run", "--user", "--wait", "--collect", "--unit="+unit, "--property=SyslogLevel=err", "--", "/usr/bin/echo", "private-fixture-message"); err != nil {
		t.Fatal(err)
	}
	poll := 10
	waitHost(t, "selected journal entry persisted", 4*time.Second, func() bool {
		poll += 6
		if err := e.sampleMonitors(ctx, start.Add(time.Duration(poll)*time.Second)); err != nil {
			t.Error(err)
			return false
		}
		var n int
		e.db.QueryRow("SELECT count(*) FROM events WHERE source='monitor:logs'").Scan(&n)
		return n == 1
	})
	paths := e.paths
	e.Close()
	reopened, err := Open(paths)
	if err != nil {
		t.Fatal(err)
	}
	e = reopened
	t.Cleanup(func() { reopened.Close() })
	if err := e.sampleMonitors(ctx, start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var failure string
	e.db.QueryRow("SELECT error FROM journal_state WHERE id='logs'").Scan(&failure)
	if failure != "" {
		t.Fatal("persisted journal cursor could not resume", failure)
	}
	var count int
	e.db.QueryRow("SELECT count(*) FROM events WHERE source='monitor:logs'").Scan(&count)
	if count != 1 {
		t.Fatal("host cursor duplicated entry")
	}
	var payload string
	e.db.QueryRow("SELECT payload FROM events WHERE source='monitor:logs'").Scan(&payload)
	if strings.Contains(payload, "private-fixture-message") {
		t.Fatal("host journal message leaked")
	}
	if _, err := readJournal(ctx, m, journalState{Cursor: "invalid-fixture-cursor"}); err == nil {
		t.Fatal("invalid cursor accepted")
	}
}

func TestHostTemperatureSensors(t *testing.T) {
	hostIsolation(t)
	paths, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	hwmon, _ := filepath.Glob("/sys/class/hwmon/hwmon*/temp*_input")
	paths = append(paths, hwmon...)
	for _, path := range paths {
		value, err := extendedMetric(context.Background(), Monitor{Metric: "temperature", Path: path}, time.Now())
		if err == nil {
			t.Logf("available hardware sensor: %.1f degrees Celsius", value)
			return
		}
	}
	t.Skip("host has no accessible temperature sensor; unavailable behavior remains explicit")
}

func TestMonitorRevocationBetweenSampleAndCommit(t *testing.T) {
	e := testEngine(t)
	ctx := context.Background()
	c := notificationConfig()
	m := Monitor{ID: "file", Metric: "file_exists", Path: "/tmp/quatrro-fixture", Threshold: 50, Recovery: 75, Interval: 5, Cooldown: 5, Enabled: true}
	c.Monitors = []Monitor{m}
	c.Flows[0].Source = "monitor:file"
	activateTest(t, e, c)
	state := monitorState{Hash: monitorHash(c, m), Phase: "alert", Next: 105, Value: 0}
	e.revoke(ctx, []byte(`{"scope":"monitor:file"}`))
	if err := e.commitMonitor(ctx, m, state, "alert", time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	var count int
	e.db.QueryRow("SELECT count(*) FROM events").Scan(&count)
	if count != 0 {
		t.Fatal("revoked sample committed event")
	}
}
