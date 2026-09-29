package broker

import (
	"bytes"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

// PeerIdentity pins the socket creator with a kernel pidfd. There is no fallback
// to looking up an unpinned PID if SO_PEERPIDFD is unavailable.
type PeerIdentity struct {
	mu    sync.Mutex
	peer  Peer
	pidfd int
	proc  *os.File
}

func IdentifyPeer(conn *net.UnixConn) (*PeerIdentity, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return nil, ErrDenied
	}
	var cred *unix.Ucred
	pidfd := -1
	var socketErr error
	err = raw.Control(func(fd uintptr) {
		cred, socketErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if socketErr == nil {
			pidfd, socketErr = unix.GetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_PEERPIDFD)
		}
	})
	if err != nil || socketErr != nil || cred == nil || cred.Pid <= 1 {
		if pidfd >= 0 {
			unix.Close(pidfd)
		}
		return nil, ErrDenied
	}
	unix.CloseOnExec(pidfd)
	fd, err := unix.Open("/proc/"+strconv.Itoa(int(cred.Pid)), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		unix.Close(pidfd)
		return nil, ErrDenied
	}
	identity := &PeerIdentity{peer: Peer{PID: int(cred.Pid), UID: cred.Uid}, pidfd: pidfd, proc: os.NewFile(uintptr(fd), "broker-peer")}
	start, err := identity.inspect()
	if err != nil {
		identity.Close()
		return nil, ErrDenied
	}
	identity.peer.StartTime = start
	return identity, nil
}

func (p *PeerIdentity) Peer() Peer { p.mu.Lock(); defer p.mu.Unlock(); return p.peer }
func (p *PeerIdentity) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pidfd >= 0 {
		unix.Close(p.pidfd)
		p.pidfd = -1
	}
	if p.proc != nil {
		p.proc.Close()
		p.proc = nil
	}
	return nil
}
func (p *PeerIdentity) Revalidate() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	start, err := p.inspect()
	if err != nil || start != p.peer.StartTime {
		return ErrDenied
	}
	return nil
}
func (p *PeerIdentity) alive() bool {
	if p.pidfd < 0 || p.proc == nil {
		return false
	}
	fds := []unix.PollFd{{Fd: int32(p.pidfd), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 0)
	return err == nil && n == 0 && fds[0].Revents == 0
}
func (p *PeerIdentity) inspect() (uint64, error) {
	if !p.alive() {
		return 0, ErrDenied
	}
	read := func(name string) ([]byte, error) {
		fd, err := unix.Openat(int(p.proc.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if err != nil {
			return nil, ErrDenied
		}
		f := os.NewFile(uintptr(fd), "broker-peer-metadata")
		defer f.Close()
		raw, err := io.ReadAll(io.LimitReader(f, 65537))
		if err != nil || len(raw) > 65536 {
			return nil, ErrDenied
		}
		return raw, nil
	}
	stat, err := read("stat")
	if err != nil {
		return 0, ErrDenied
	}
	start, err := processStartTime(stat, p.peer.PID)
	if err != nil {
		return 0, ErrDenied
	}
	status, err := read("status")
	if err != nil {
		return 0, ErrDenied
	}
	if !processUIDMatches(status, p.peer.UID) || !p.alive() {
		return 0, ErrDenied
	}
	return start, nil
}

func processStartTime(raw []byte, pid int) (uint64, error) {
	open := bytes.IndexByte(raw, '(')
	close := bytes.LastIndex(raw, []byte(") "))
	if open < 1 || close <= open {
		return 0, ErrDenied
	}
	number, err := strconv.Atoi(strings.TrimSpace(string(raw[:open])))
	if err != nil || number != pid {
		return 0, ErrDenied
	}
	fields := strings.Fields(string(raw[close+2:]))
	if len(fields) < 20 {
		return 0, ErrDenied
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || start == 0 {
		return 0, ErrDenied
	}
	return start, nil
}
func processUIDMatches(raw []byte, uid uint32) bool {
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(strings.TrimPrefix(line, "Uid:"))
			if len(fields) != 4 {
				return false
			}
			for _, field := range fields {
				v, err := strconv.ParseUint(field, 10, 32)
				if err != nil || uint32(v) != uid {
					return false
				}
			}
			return true
		}
	}
	return false
}
