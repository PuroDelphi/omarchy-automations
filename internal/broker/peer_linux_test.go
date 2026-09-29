package broker

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestBrokerPeerKernelIdentity(t *testing.T) {
	pair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(pair[1])
	f := os.NewFile(uintptr(pair[0]), "test-socket")
	conn, err := net.FileConn(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	identity, err := IdentifyPeer(conn.(*net.UnixConn))
	if err != nil {
		t.Fatal("kernel peer identity unavailable", err)
	}
	defer identity.Close()
	peer := identity.Peer()
	if peer.PID != os.Getpid() || peer.UID != uint32(os.Getuid()) || peer.StartTime == 0 {
		t.Fatal("incorrect kernel subject")
	}
	if err = identity.Revalidate(); err != nil {
		t.Fatal(err)
	}
	identity.Close()
	if identity.Revalidate() != ErrDenied {
		t.Fatal("closed identity accepted")
	}
}

func TestBrokerProcessMetadataParsing(t *testing.T) {
	// comm may contain spaces, parentheses and newlines. Only the final ') '
	// separates it from numeric fields; naive whitespace splitting is unsafe.
	fields := append([]string{"S"}, strings.Fields(strings.Repeat("1 ", 18))...)
	fields = append(fields, "456")
	raw := []byte("123 (name ) with\n(parentheses)) " + strings.Join(fields, " "))
	if start, err := processStartTime(raw, 123); err != nil || start != 456 {
		t.Fatal("comm parsing failed", err)
	}
	if _, err := processStartTime(raw, 124); err != ErrDenied {
		t.Fatal("PID mismatch accepted")
	}
	for _, raw := range []string{"123 (name) S", "123 name S", "123 (name) " + strings.Repeat("x ", 20)} {
		if _, err := processStartTime([]byte(raw), 123); err != ErrDenied {
			t.Fatal("invalid proc stat accepted")
		}
	}
	if !processUIDMatches([]byte(fmt.Sprintf("Name:\tx\nUid:\t%d\t%d\t%d\t%d\n", 1000, 1000, 1000, 1000)), 1000) {
		t.Fatal("valid UID rejected")
	}
	for _, status := range []string{"Uid:\t1000\t0\t1000\t1000", "Uid:\t1000\t1000", "Uid:\t-1\t-1\t-1\t-1", "Name:\tx"} {
		if processUIDMatches([]byte(status), 1000) {
			t.Fatal("changed/incomplete UID accepted")
		}
	}
}

func TestBrokerPeerChildHelper(t *testing.T) {
	path := os.Getenv("QUATRRO_TEST_PEER_SOCKET")
	if path == "" {
		return
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		os.Exit(2)
	}
	defer conn.Close()
	if _, err = conn.Write([]byte("ready")); err != nil {
		os.Exit(3)
	}
	var b [1]byte
	conn.Read(b[:])
}

func TestBrokerPeerRejectsExitedProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peer.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	child := exec.Command(os.Args[0], "-test.run=^TestBrokerPeerChildHelper$")
	child.Env = append(os.Environ(), "QUATRRO_TEST_PEER_SOCKET="+path)
	if err = child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { child.Process.Kill(); child.Wait() }()
	listener.SetDeadline(time.Now().Add(5 * time.Second))
	conn, err := listener.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var ready [5]byte
	if _, err = io.ReadFull(conn, ready[:]); err != nil {
		t.Fatal(err)
	}
	identity, err := IdentifyPeer(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer identity.Close()
	if identity.Peer().PID != child.Process.Pid || identity.Revalidate() != nil {
		t.Fatal("child identity mismatch")
	}
	if err = child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	child.Wait()
	if identity.Revalidate() != ErrDenied {
		t.Fatal("dead peer authorized")
	}
}
