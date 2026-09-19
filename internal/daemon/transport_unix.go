//go:build !windows
// +build !windows

package daemon

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/inoki/sgreen/internal/pty"
)

// socketDialTimeout bounds transport dials.
const socketDialTimeout = 2 * time.Second

// listenTransport serves the session socket at path (Unix domain socket;
// filesystem permissions carry the access control, no token needed).
func listenTransport(path string) (net.Listener, string, error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	return ln, "", err
}

// dialTransport connects to a session socket.
func dialTransport(path string) (net.Conn, error) {
	return net.DialTimeout("unix", path, 2*socketDialTimeout)
}

// initialPTYFromEnv rebuilds the initial window's PTY master from the fd
// the creating process handed over.
func initialPTYFromEnv() (*pty.PTYProcess, error) {
	fd, err := strconv.Atoi(os.Getenv("SGREEN_HOLD_FD"))
	if err != nil || fd <= 0 {
		return nil, fmt.Errorf("invalid SGREEN_HOLD_FD=%q", os.Getenv("SGREEN_HOLD_FD"))
	}
	master := os.NewFile(uintptr(fd), "sgreen-pty-master")
	if master == nil {
		return nil, fmt.Errorf("failed to open master fd=%d", fd)
	}
	return &pty.PTYProcess{Pty: master}, nil
}
