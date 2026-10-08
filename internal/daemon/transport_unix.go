//go:build !windows
// +build !windows

package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/inoki/sgreen/internal/pty"
)

// socketDialTimeout bounds transport dials.
const socketDialTimeout = 2 * time.Second

// listenTransport serves the session socket at path (Unix domain socket;
// filesystem permissions carry the access control, no token needed).
// When the path exceeds the platform's sun_path limit (104 bytes on
// macOS, e.g. a long SCREENDIR), it binds under TMPDIR instead and
// records "unix <real-path>" into path as an endpoint file.
func listenTransport(path string) (net.Listener, string, error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err == nil {
		return ln, "", nil
	}
	if !errors.Is(err, syscall.EINVAL) {
		return nil, "", err
	}
	fallback := filepath.Join(os.TempDir(),
		fmt.Sprintf("sgreen-%s-%d.sock", sanitizeEndpointName(filepath.Base(path)), os.Getpid()))
	ln, err = net.Listen("unix", fallback)
	if err != nil {
		return nil, "", err
	}
	if werr := os.WriteFile(path, []byte("unix "+fallback+"\n"), 0600); werr != nil {
		_ = ln.Close()
		return nil, "", werr
	}
	return ln, "", nil
}

// dialTransport connects to a session socket: either a real socket at
// path, or the address recorded in an endpoint file written by the
// fallback above.
func dialTransport(path string) (net.Conn, error) {
	if fi, err := os.Stat(path); err == nil && fi.Mode().IsRegular() {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fields := strings.Fields(string(data))
		if len(fields) == 2 && fields[0] == "unix" {
			return net.DialTimeout("unix", fields[1], socketDialTimeout)
		}
		return nil, fmt.Errorf("malformed session endpoint file %s", path)
	}
	return net.DialTimeout("unix", path, socketDialTimeout)
}

// sanitizeEndpointName keeps the fallback socket filename filesystem- and
// sun_path-friendly.
func sanitizeEndpointName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := b.String()
	if len(out) > 32 {
		out = out[:32]
	}
	return out
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
