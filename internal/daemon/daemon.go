//go:build !windows
// +build !windows

// Package daemon implements the per-session relay daemon.
//
// A session daemon is forked when a session is created. It is the single
// owner of the session's PTY master file descriptor: it relays data between
// the master and any number of attached clients over a Unix socket, and
// maintains the "attached" flag in the session file. This mirrors the GNU
// screen model, where a resident screen process owns the session and
// clients talk to it through a socket.
package daemon

import (
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/inoki/sgreen/internal/session"
)

const (
	// maxProtocolLine bounds the first-line length accepted on new
	// connections so a bogus client cannot grow memory unbounded.
	maxProtocolLine = 64
)

// Supported reports whether session daemons are available on this platform.
func Supported() bool { return true }

// RunFromEnv starts a session daemon when this process was forked for that
// purpose (SGREEN_SESSION_DAEMON=1). It returns true if the daemon ran.
func RunFromEnv() bool {
	if os.Getenv("SGREEN_SESSION_DAEMON") != "1" {
		return false
	}

	fd, err := strconv.Atoi(os.Getenv("SGREEN_HOLD_FD"))
	if err != nil || fd <= 0 {
		log.Printf("sgreen-daemon: invalid SGREEN_HOLD_FD=%q", os.Getenv("SGREEN_HOLD_FD"))
		return true
	}
	master := os.NewFile(uintptr(fd), "sgreen-pty-master")
	if master == nil {
		log.Printf("sgreen-daemon: failed to open master fd=%d", fd)
		return true
	}

	sessionFile := os.Getenv("SGREEN_SESSION_FILE")
	sessionID := os.Getenv("SGREEN_SESSION_ID")
	socketPath := os.Getenv("SGREEN_SESSION_SOCK")

	if logPath := os.Getenv("SGREEN_DAEMON_LOG"); logPath != "" {
		if f, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
			log.SetOutput(f)
		}
	}
	log.Printf("sgreen-daemon: forked file=%q id=%q sock=%q", sessionFile, sessionID, socketPath)
	sessionPid := 0
	if v := os.Getenv("SGREEN_SESSION_PID"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			sessionPid = n
		}
	}

	var ready *os.File
	if readyFD, err := strconv.Atoi(os.Getenv("SGREEN_READY_FD")); err == nil && readyFD > 0 {
		ready = os.NewFile(uintptr(readyFD), "sgreen-daemon-ready")
	}

	Run(master, sessionFile, sessionID, socketPath, sessionPid, ready)
	return true
}

// state tracks the daemon's attachers and serializes session-file updates.
type state struct {
	mu         sync.Mutex
	attachers  map[net.Conn]struct{}
	sessionObj *session.Session
	// shutDown is set once cleanup starts; attachers unregistering after
	// that must not write the session file back (it would resurrect the
	// file cleanup just removed).
	shutDown bool
}

// updateSession persists the daemon-maintained runtime fields. attached is
// the authoritative attach state; daemonPid of 0 keeps the existing value.
func (st *state) updateSession(attached bool, daemonPid int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.shutDown || st.sessionObj == nil {
		return
	}
	if st.sessionObj.Attached == attached && daemonPid == 0 {
		return
	}
	st.sessionObj.Attached = attached
	if daemonPid != 0 {
		st.sessionObj.DaemonPid = daemonPid
	}
	_ = st.sessionObj.Save()
}

// Run serves a session until the session program exits or QUIT is received.
// It owns master for the lifetime of the call and removes the socket and
// session file on exit. Run never returns normally except on listen
// failure; all other exits go through cleanup + os.Exit.
func Run(master *os.File, sessionFile, sessionID, socketPath string, sessionPid int, ready *os.File) {
	logger := log.New(log.Writer(), "sgreen-daemon: ", log.LstdFlags)

	// Load the session file the creating process wrote, then take ownership
	// of the runtime fields.
	sess, err := session.LoadFromFile(sessionFile, sessionID)
	if err != nil {
		logger.Printf("failed to load session file %s: %v", sessionFile, err)
		sess = &session.Session{ID: sessionID}
	} else {
		logger.Printf("loaded session %q daemon_pid=%d attached=%v", sess.ID, sess.DaemonPid, sess.Attached)
	}

	st := &state{
		attachers:  make(map[net.Conn]struct{}),
		sessionObj: sess,
	}

	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Printf("failed to listen on %s: %v", socketPath, err)
		// The parent is waiting on the ready pipe; unblock it so it can
		// fall back instead of hanging.
		signalReady(ready)
		return
	}

	// Take ownership of the runtime fields, then unblock the parent.
	st.updateSession(false, os.Getpid())
	signalReady(ready)

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			logger.Printf("cleanup: starting")
			st.mu.Lock()
			st.shutDown = true
			for conn := range st.attachers {
				_ = conn.Close()
			}
			st.mu.Unlock()
			_ = ln.Close()
			if err := os.Remove(socketPath); err != nil {
				logger.Printf("cleanup: remove socket: %v", err)
			}
			if err := session.RemoveFile(sessionFile, sessionID); err != nil {
				logger.Printf("cleanup: remove session file: %v", err)
			}
			_ = master.Close()
			killSessionProcess(sessionPid)
			logger.Printf("cleanup: done")
		})
	}

	// Exit paths: session program exited (master EOF), QUIT command, signal.
	sigChan := make(chan os.Signal, 1)
	notifySignals(sigChan)
	go func() {
		for range sigChan {
			cleanup()
			os.Exit(0)
		}
	}()

	go func() {
		relayMasterToAttachers(master, st, logger)
		cleanup()
		os.Exit(0)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// The listener was closed by a cleanup path (QUIT, master EOF,
			// signal). That path finishes cleanup and calls os.Exit itself;
			// returning here would race main()'s exit and could kill the
			// process before cleanup removed the session files.
			select {}
		}
		go handleConn(conn, master, st, sessionPid, logger, cleanup)
	}
}

func signalReady(ready *os.File) {
	if ready == nil {
		return
	}
	_, _ = ready.Write([]byte("ready\n"))
	_ = ready.Close()
}

// relayMasterToAttachers fans PTY output out to every attached client. It
// returns when the master is closed or errors (session program exited).
func relayMasterToAttachers(master *os.File, st *state, logger *log.Logger) {
	buf := make([]byte, 32*1024)
	for {
		n, err := master.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			st.mu.Lock()
			for conn := range st.attachers {
				if _, werr := conn.Write(data); werr != nil {
					_ = conn.Close()
					delete(st.attachers, conn)
				}
			}
			remaining := len(st.attachers)
			st.mu.Unlock()
			if remaining == 0 {
				st.updateSession(false, 0)
			}
		}
		if err != nil {
			logger.Printf("master read end: %v", err)
			return
		}
	}
}

// handleConn reads the protocol first line and dispatches to ATTACH relay
// or CONTROL command handling.
func handleConn(conn net.Conn, master *os.File, st *state, sessionPid int, logger *log.Logger, onQuit func()) {
	line, err := readLine(conn)
	if err != nil {
		_ = conn.Close()
		return
	}

	switch strings.TrimSpace(line) {
	case "ATTACH":
		serveAttach(conn, master, st, logger)
	case "CONTROL":
		serveControl(conn, master, st, logger, onQuit)
	default:
		_, _ = conn.Write([]byte("ERR unknown mode\n"))
		_ = conn.Close()
	}
}

// readLine reads a newline-terminated line byte by byte so no raw payload
// following the handshake line is consumed from the socket buffer.
func readLine(conn net.Conn) (string, error) {
	var b strings.Builder
	one := make([]byte, 1)
	for b.Len() < maxProtocolLine {
		_, err := conn.Read(one)
		if err != nil {
			return "", err
		}
		if one[0] == '\n' {
			return b.String(), nil
		}
		if one[0] != '\r' {
			b.WriteByte(one[0])
		}
	}
	return b.String(), nil
}

// serveAttach relays one attached client until the connection closes.
func serveAttach(conn net.Conn, master *os.File, st *state, logger *log.Logger) {
	st.mu.Lock()
	st.attachers[conn] = struct{}{}
	st.mu.Unlock()
	st.updateSession(true, 0)
	logger.Printf("attacher registered: %s", conn.RemoteAddr())

	defer func() {
		st.mu.Lock()
		delete(st.attachers, conn)
		remaining := len(st.attachers)
		st.mu.Unlock()
		if remaining == 0 {
			st.updateSession(false, 0)
		}
		_ = conn.Close()
	}()

	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			if _, werr := master.Write(buf[:n]); werr != nil {
				logger.Printf("master write: %v", werr)
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// serveControl executes control commands, one per line, until EOF.
func serveControl(conn net.Conn, master *os.File, st *state, logger *log.Logger, onQuit func()) {
	defer func() { _ = conn.Close() }()
	for {
		line, err := readLine(conn)
		if err != nil {
			return
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "STATUS":
			st.mu.Lock()
			n := len(st.attachers)
			st.mu.Unlock()
			if n > 0 {
				_, _ = fmt.Fprintf(conn, "OK ATTACHED %d\n", n)
			} else {
				_, _ = conn.Write([]byte("OK DETACHED\n"))
			}
		case "DETACH", "POWER-DETACH":
			kickAttachers(st)
			_, _ = conn.Write([]byte("OK\n"))
		case "RESIZE":
			if len(parts) != 3 {
				_, _ = conn.Write([]byte("ERR usage: RESIZE rows cols\n"))
				continue
			}
			rows, err1 := strconv.Atoi(parts[1])
			cols, err2 := strconv.Atoi(parts[2])
			if err1 != nil || err2 != nil || rows <= 0 || cols <= 0 {
				_, _ = conn.Write([]byte("ERR bad size\n"))
				continue
			}
			logger.Printf("resize request rows=%d cols=%d", rows, cols)
			if err := setMasterSize(master, uint16(rows), uint16(cols)); err != nil {
				_, _ = fmt.Fprintf(conn, "ERR %v\n", err)
				continue
			}
			_, _ = conn.Write([]byte("OK\n"))
		case "STUFF":
			if len(parts) != 2 {
				_, _ = conn.Write([]byte("ERR usage: STUFF base64data\n"))
				continue
			}
			data, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				_, _ = conn.Write([]byte("ERR bad base64\n"))
				continue
			}
			if _, err := master.Write(data); err != nil {
				_, _ = fmt.Fprintf(conn, "ERR %v\n", err)
				continue
			}
			_, _ = conn.Write([]byte("OK\n"))
		case "QUIT":
			_, _ = conn.Write([]byte("OK\n"))
			logger.Printf("QUIT received")
			onQuit()
			// Exit here so the cleanup above (file removal) completes
			// atomically instead of racing main()'s return.
			os.Exit(0)
		default:
			_, _ = fmt.Fprintf(conn, "ERR unknown command %q\n", parts[0])
		}
	}
}

func kickAttachers(st *state) {
	st.mu.Lock()
	conns := make([]net.Conn, 0, len(st.attachers))
	for conn := range st.attachers {
		conns = append(conns, conn)
		delete(st.attachers, conn)
	}
	st.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close()
	}
	st.updateSession(false, 0)
}
