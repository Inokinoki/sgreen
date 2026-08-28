//go:build !windows
// +build !windows

package daemon

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// connectControl opens a short-lived control connection to a session daemon.
func connectControl(socketPath string) (net.Conn, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write([]byte("CONTROL\n")); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// controlRoundtrip sends one command and returns the first response line.
func controlRoundtrip(socketPath, command string) (string, error) {
	conn, err := connectControl(socketPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte(command + "\n")); err != nil {
		return "", err
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, err := readLine(conn)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Status describes a session daemon's live state.
type Status struct {
	Attached bool
	Count    int
}

// QueryStatus asks a session daemon whether anyone is attached.
func QueryStatus(socketPath string) (*Status, error) {
	resp, err := controlRoundtrip(socketPath, "STATUS")
	if err != nil {
		return nil, err
	}
	switch {
	case resp == "OK DETACHED":
		return &Status{Attached: false}, nil
	case strings.HasPrefix(resp, "OK ATTACHED"):
		var n int
		if _, err := fmt.Sscanf(resp, "OK ATTACHED %d", &n); err != nil || n < 1 {
			n = 1
		}
		return &Status{Attached: true, Count: n}, nil
	default:
		return nil, fmt.Errorf("daemon status: %s", resp)
	}
}

// SendDetach asks the daemon to detach the currently attached clients
// (power=true uses the power-detach form).
func SendDetach(socketPath string, power bool) error {
	cmd := "DETACH"
	if power {
		cmd = "POWER-DETACH"
	}
	resp, err := controlRoundtrip(socketPath, cmd)
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("daemon %s: %s", cmd, resp)
	}
	return nil
}

// SendQuit terminates the session (daemon kills the program and cleans up).
func SendQuit(socketPath string) error {
	resp, err := controlRoundtrip(socketPath, "QUIT")
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("daemon quit: %s", resp)
	}
	return nil
}

// SendStuff writes data to the session program's input as if typed.
func SendStuff(socketPath string, data string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(data))
	resp, err := controlRoundtrip(socketPath, "STUFF "+encoded)
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("daemon stuff: %s", resp)
	}
	return nil
}

// SendResize applies a terminal window size to the session PTY.
func SendResize(socketPath string, rows, cols uint16) error {
	resp, err := controlRoundtrip(socketPath, fmt.Sprintf("RESIZE %d %d", rows, cols))
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("daemon resize: %s", resp)
	}
	return nil
}

// OpenAttach opens an ATTACH data connection to the session daemon and
// performs the handshake. The returned connection carries raw PTY data.
func OpenAttach(socketPath string) (net.Conn, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write([]byte("ATTACH\n")); err != nil {
		_ = conn.Close()
		return nil, err
	}
	// The daemon does not acknowledge ATTACH; relay starts immediately.
	// Validate liveness with a STATUS probe on a separate connection so a
	// dead daemon is detected before the caller enters raw mode.
	status, err := QueryStatus(socketPath)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = status
	return conn, nil
}

// ErrNoDaemon indicates no session daemon is listening on the socket.
var ErrNoDaemon = errors.New("no session daemon")
