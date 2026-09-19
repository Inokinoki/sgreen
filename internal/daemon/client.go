package daemon

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/inoki/sgreen/internal/pty"
	"github.com/inoki/sgreen/internal/session"
)

// connectControl opens a short-lived control connection to a session daemon.
func connectControl(socketPath string) (net.Conn, error) {
	conn, err := dialTransport(socketPath)
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
	Current  int
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
		var n, cur int
		if _, err := fmt.Sscanf(resp, "OK ATTACHED %d %d", &n, &cur); err != nil || n < 1 {
			n = 1
		}
		return &Status{Attached: true, Count: n, Current: cur}, nil
	case strings.HasPrefix(resp, "OK DETACHED "):
		var cur int
		_, _ = fmt.Sscanf(resp, "OK DETACHED %d", &cur)
		return &Status{Attached: false, Current: cur}, nil
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

// SendQuit terminates the session (daemon kills the programs and cleans up).
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

// SendStuff writes data to the current window's program as if typed.
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

// SendResizeWindow applies a terminal window size to one window's PTY.
func SendResizeWindow(socketPath string, winID int, rows, cols uint16) error {
	command := fmt.Sprintf("RESIZE %d %d %d", winID, rows, cols)
	resp, err := controlRoundtrip(socketPath, command)
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("daemon resize: %s", resp)
	}
	return nil
}

// OpenAttach opens an ATTACH data connection for one window and performs
// the handshake. The returned connection carries raw PTY data.
func OpenAttach(socketPath string, winID int) (net.Conn, error) {
	conn, err := dialTransport(socketPath)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Write([]byte(fmt.Sprintf("ATTACH %d\n", winID))); err != nil {
		_ = conn.Close()
		return nil, err
	}
	// The daemon does not acknowledge ATTACH; relay starts immediately.
	// Validate liveness with a STATUS probe on a separate connection so a
	// dead daemon is detected before the caller enters raw mode.
	if _, err := QueryStatus(socketPath); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// ErrNoDaemon indicates no session daemon is listening on the socket.
var ErrNoDaemon = errors.New("no session daemon")

// Controller implements session.WindowController against a session daemon.
type Controller struct {
	socketPath string
}

// NewController returns a window controller talking to the daemon at
// socketPath.
func NewController(socketPath string) *Controller {
	return &Controller{socketPath: socketPath}
}

// CreateWindow implements session.WindowController.
func (c *Controller) CreateWindow(cmdPath string, args []string, term string) (int, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return 0, err
	}
	// "-" is the empty-token placeholder: strings.Fields would otherwise
	// drop an empty base64 string and break the argument count.
	if term == "" {
		term = "-"
	}
	command := fmt.Sprintf("NEWWINDOW %s %s %s",
		base64.StdEncoding.EncodeToString([]byte(cmdPath)),
		base64.StdEncoding.EncodeToString(argsJSON),
		base64.StdEncoding.EncodeToString([]byte(term)))
	resp, err := controlRoundtrip(c.socketPath, command)
	if err != nil {
		return 0, err
	}
	id, ok := parseOKInt(resp)
	if !ok {
		return 0, fmt.Errorf("daemon newwindow: %s", resp)
	}
	return id, nil
}

// SwitchWindow implements session.WindowController.
func (c *Controller) SwitchWindow(op string, arg string) (int, error) {
	cmd := strings.ToUpper(op)
	if cmd == "SELECT" {
		cmd = "SELECT " + arg
	}
	resp, err := controlRoundtrip(c.socketPath, cmd)
	if err != nil {
		return 0, err
	}
	id, ok := parseOKInt(resp)
	if !ok {
		return 0, fmt.Errorf("daemon switch: %s", resp)
	}
	return id, nil
}

// KillWindow implements session.WindowController.
func (c *Controller) KillWindow(winID int) error {
	resp, err := controlRoundtrip(c.socketPath, fmt.Sprintf("KILLWINDOW %d", winID))
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("daemon killwindow: %s", resp)
	}
	return nil
}

// SetTitle implements session.WindowController.
func (c *Controller) SetTitle(winID int, title string) error {
	command := fmt.Sprintf("SETTITLE %d %s", winID, base64.StdEncoding.EncodeToString([]byte(title)))
	resp, err := controlRoundtrip(c.socketPath, command)
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("daemon settitle: %s", resp)
	}
	return nil
}

// OpenWindowProcess implements session.WindowController: it returns a
// relay endpoint for the window's PTY with control hooks bound to that
// window.
func (c *Controller) OpenWindowProcess(win *session.Window) (*pty.PTYProcess, error) {
	if win == nil {
		return nil, fmt.Errorf("nil window")
	}
	conn, err := OpenAttach(c.socketPath, win.ID)
	if err != nil {
		return nil, err
	}
	remote := pty.NewRemote(conn, pty.RemoteControl{
		Resize: func(rows, cols uint16) error {
			return SendResizeWindow(c.socketPath, win.ID, rows, cols)
		},
		Alive: func() bool {
			_, err := QueryStatus(c.socketPath)
			return err == nil
		},
		Quit: func() error {
			return SendQuit(c.socketPath)
		},
	})
	return pty.NewRemoteProcess(remote, win.PtsPath), nil
}

// SendKillCurrent kills the session's current window. Killing the last
// window ends the session (GNU semantics).
func SendKillCurrent(socketPath string) error {
	cur, err := currentWindow(socketPath)
	if err != nil {
		return err
	}
	ctrl := &Controller{socketPath: socketPath}
	return ctrl.KillWindow(cur)
}

// SendTitleCurrent renames the session's current window.
func SendTitleCurrent(socketPath string, title string) error {
	cur, err := currentWindow(socketPath)
	if err != nil {
		return err
	}
	ctrl := &Controller{socketPath: socketPath}
	return ctrl.SetTitle(cur, title)
}

// currentWindow asks the daemon which window is current.
func currentWindow(socketPath string) (int, error) {
	st, err := QueryStatus(socketPath)
	if err != nil {
		return 0, err
	}
	return st.Current, nil
}

func parseOKInt(resp string) (int, bool) {
	if !strings.HasPrefix(resp, "OK ") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(resp[3:]))
	if err != nil {
		return 0, false
	}
	return n, true
}
