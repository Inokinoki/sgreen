//go:build windows
// +build windows

package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/inoki/sgreen/internal/pty"
)

// socketDialTimeout bounds transport dials.
const socketDialTimeout = 2 * time.Second

// listenTransport serves the session on a loopback TCP port and records
// "127.0.0.1:<port> <token>" into sockPath. Named pipes would avoid the
// token, but stdlib has no listener for them; loopback TCP plus a shared
// secret keeps the dependency surface empty.
func listenTransport(sockPath string) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		_ = ln.Close()
		return nil, "", err
	}
	token := hex.EncodeToString(raw)
	content := fmt.Sprintf("%s %s\n", ln.Addr().String(), token)
	if err := os.WriteFile(sockPath, []byte(content), 0600); err != nil {
		_ = ln.Close()
		return nil, "", err
	}
	return ln, token, nil
}

// dialTransport reads the endpoint file and connects, performing the AUTH
// handshake for token-protected transports.
func dialTransport(sockPath string) (net.Conn, error) {
	data, err := os.ReadFile(sockPath)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return nil, fmt.Errorf("malformed session endpoint file %s", sockPath)
	}
	conn, err := net.DialTimeout("tcp", fields[0], socketDialTimeout)
	if err != nil {
		return nil, err
	}
	if len(fields) >= 2 {
		if _, err := conn.Write([]byte("AUTH " + fields[1] + "\n")); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

// initialPTYFromEnv returns nil on Windows: no PTY is handed over, the
// daemon bootstraps the initial window from the command environment.
func initialPTYFromEnv() (*pty.PTYProcess, error) {
	return nil, nil
}

// cleanupTransport removes the endpoint file (the TCP listener closes
// with the daemon).
func cleanupTransport(path string) {
	_ = os.Remove(path)
}
