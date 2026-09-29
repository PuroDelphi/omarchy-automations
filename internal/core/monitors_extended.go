package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func lowMetric(metric string) bool {
	switch metric {
	case "disk", "battery", "service", "process", "file_exists", "connectivity":
		return true
	}
	return false
}

func metricUnit(metric string) string {
	switch metric {
	case "temperature":
		return "°C"
	case "file_age":
		return "s"
	case "file_size":
		return "bytes"
	}
	return "%"
}

func validateMonitor(m Monitor, seen map[string]bool) error {
	minimum, maximum := 0.0, 100.0
	switch m.Metric {
	case "journal":
		if m.Priority < 0 || m.Priority > 7 {
			return errors.New("journal priority must be 0..7")
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.@-]{0,127}\.service$`).MatchString(m.Unit) {
			return errors.New("journal requires an explicit user service unit")
		}
		return nil
	case "cpu", "memory", "disk", "battery", "service", "process", "file_exists", "connectivity":
	case "temperature":
		minimum, maximum = -273.15, 1000
	case "file_age":
		maximum = 31536000
	case "file_size":
		maximum = 1e15
	default:
		return errors.New("unknown monitor metric")
	}
	if math.IsNaN(m.Threshold) || math.IsNaN(m.Recovery) || m.Threshold < minimum || m.Recovery < minimum || m.Threshold > maximum || m.Recovery > maximum {
		return fmt.Errorf("thresholds outside range %g..%g %s", minimum, maximum, metricUnit(m.Metric))
	}
	if (lowMetric(m.Metric) && m.Recovery <= m.Threshold) || (!lowMetric(m.Metric) && m.Recovery >= m.Threshold) {
		return errors.New("recovery must cross threshold in opposite direction")
	}
	switch m.Metric {
	case "disk":
		if m.Path != "" && !filepath.IsAbs(m.Path) {
			return errors.New("disk path must be absolute")
		}
	case "service":
		if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.@-]{0,127}\.service$`).MatchString(m.Unit) {
			return errors.New("invalid user unit")
		}
	case "process", "file_exists", "file_age", "file_size":
		if !filepath.IsAbs(m.Path) || filepath.Clean(m.Path) != m.Path || strings.ContainsRune(m.Path, 0) || len(m.Path) > 4096 {
			return errors.New("approved path must be absolute and canonical")
		}
	case "temperature":
		if !regexp.MustCompile(`^/sys/class/(thermal/thermal_zone[0-9]+/temp|hwmon/hwmon[0-9]+/temp[0-9]+_input)$`).MatchString(m.Path) {
			return errors.New("choose an explicit thermal or hwmon temperature sensor")
		}
	case "connectivity":
		if !seen["destination:"+m.Destination] {
			return errors.New("connectivity needs a registered HTTPS destination")
		}
	}
	return nil
}

func monitorDestination(c Config, m Monitor) (Destination, bool) {
	for _, d := range c.Destinations {
		if d.ID == m.Destination {
			return d, true
		}
	}
	return Destination{}, false
}

func monitorHash(c Config, m Monitor) string {
	if m.Metric != "connectivity" {
		return canonicalHash(m)
	}
	d, _ := monitorDestination(c, m)
	return canonicalHash(struct {
		Monitor     Monitor
		Destination Destination
	}{m, d})
}

func extendedMetric(ctx context.Context, m Monitor, now time.Time) (float64, error) {
	switch m.Metric {
	case "process":
		return processPresence(ctx, m.Path)
	case "file_exists", "file_age", "file_size":
		// O_PATH reads metadata without opening devices/FIFOs or following any
		// symlink component. No file contents are collected or exposed in events.
		fd, err := unix.Openat2(unix.AT_FDCWD, m.Path, &unix.OpenHow{Flags: unix.O_PATH | unix.O_CLOEXEC, Resolve: unix.RESOLVE_NO_SYMLINKS})
		if errors.Is(err, syscall.ENOENT) && m.Metric == "file_exists" {
			return 0, nil
		}
		if err != nil {
			return 0, errors.New("file metadata unavailable")
		}
		f := os.NewFile(uintptr(fd), "approved-monitor-file")
		defer f.Close()
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return 0, errors.New("approved path is not a regular file")
		}
		switch m.Metric {
		case "file_exists":
			return 100, nil
		case "file_size":
			return float64(info.Size()), nil
		default:
			age := now.Sub(info.ModTime()).Seconds()
			if age < 0 {
				return 0, errors.New("file modification time is in the future")
			}
			return age, nil
		}
	case "temperature":
		f, err := os.Open(m.Path)
		if err != nil {
			return 0, errors.New("sensor unavailable")
		}
		defer f.Close()
		raw, err := io.ReadAll(io.LimitReader(f, 64))
		if err != nil || len(raw) >= 64 {
			return 0, errors.New("sensor value unavailable")
		}
		return parseTemperature(raw)
	}
	return 0, errors.New("unsupported extended metric")
}

func parseTemperature(raw []byte) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < -273150 || value > 1000000 {
		return 0, errors.New("invalid temperature reading")
	}
	return value / 1000, nil
}

func processPresence(ctx context.Context, executable string) (float64, error) {
	resolved, _ := filepath.EvalSymlinks(executable)
	dir, err := os.Open("/proc")
	if err != nil {
		return 0, errors.New("process list unavailable")
	}
	defer dir.Close()
	entries, err := dir.Readdirnames(8193)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if len(entries) > 8192 {
		return 0, errors.New("process scan limit exceeded")
	}
	unreadable := false
	for _, name := range entries {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if _, err := strconv.Atoi(name); err != nil {
			continue
		}
		path := filepath.Join("/proc", name)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return 0, errors.New("process metadata unavailable")
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != uint32(os.Getuid()) {
			continue
		}
		target, err := os.Readlink(filepath.Join(path, "exe"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			unreadable = true
			continue
		}
		if path := strings.TrimSuffix(target, " (deleted)"); path == executable || (resolved != "" && path == resolved) {
			return 100, nil
		}
	}
	if unreadable {
		return 0, errors.New("some user process metadata is inaccessible; absence cannot be confirmed")
	}
	return 0, nil
}

func (e *Engine) connectivityMetric(ctx context.Context, c Config, m Monitor, client *http.Client) (float64, error) {
	d, ok := monitorDestination(c, m)
	if !ok {
		return 0, errors.New("monitor destination unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if client == nil {
		client = outboundClient(d, net.DefaultResolver)
		defer client.CloseIdleConnections()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, d.URL, nil)
	if err != nil {
		return 0, errors.New("monitor URL unavailable")
	}
	for key, value := range d.Headers {
		request.Header.Set(key, value)
	}
	if err = e.authenticateOutbound(ctx, request, d, "monitor-"+newID(), nil); err != nil {
		return 0, err
	}
	response, err := client.Do(request)
	if err != nil {
		// Policy denial is unavailable, not a failed reachability measurement.
		if errors.Is(err, errDestinationBlocked) {
			return 0, errors.New("monitor destination blocked by policy")
		}
		// Other transport failures are unsuccessful health checks.
		return 0, nil
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return 100, nil
	}
	return 0, nil
}
