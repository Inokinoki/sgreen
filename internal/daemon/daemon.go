// Package daemon implements the per-session relay daemon.
//
// A session daemon is forked when a session is created. It owns the PTY
// master of every window in the session: on Unix the initial master is
// handed over by the creating process and later windows (C-a c) are
// spawned inside the daemon; on Windows the daemon bootstraps even the
// initial program itself (no fd inheritance exists there). Clients attach
// through a transport (Unix socket / loopback TCP) and relay data for one
// window per ATTACH connection; window lifecycle operations arrive as
// control commands so the daemon's session file stays authoritative. This
// mirrors the GNU screen model, where a resident screen process owns the
// session and clients talk to it through a socket.
package daemon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inoki/sgreen/internal/pty"
	"github.com/inoki/sgreen/internal/session"
)

// Supported reports whether session daemons are available on this platform.
func Supported() bool { return true }

// RunFromEnv starts a session daemon when this process was forked for that
// purpose (SGREEN_SESSION_DAEMON=1). It returns true if the daemon ran.
func RunFromEnv() bool {
	if os.Getenv("SGREEN_SESSION_DAEMON") != "1" {
		return false
	}

	if logPath := os.Getenv("SGREEN_DAEMON_LOG"); logPath != "" {
		if f, logErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
			log.SetOutput(f)
		}
	}

	sessionFile := os.Getenv("SGREEN_SESSION_FILE")
	sessionID := os.Getenv("SGREEN_SESSION_ID")
	socketPath := os.Getenv("SGREEN_SESSION_SOCK")
	log.Printf("sgreen-daemon: forked file=%q id=%q sock=%q", sessionFile, sessionID, socketPath)

	initial, err := initialPTYFromEnv()
	if err != nil {
		log.Printf("sgreen-daemon: initial window: %v", err)
		return true
	}

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

	Run(initial, sessionFile, sessionID, socketPath, sessionPid, ready)
	return true
}

// masterConn is a window's PTY master endpoint: bidirectional relay plus
// resize. On Unix it wraps the master fd; on Windows the two ConPTY pipes.
type masterConn interface {
	io.ReadWriteCloser
	Resize(rows, cols uint16) error
}

// ptyMaster adapts a *pty.PTYProcess into a masterConn.
type ptyMaster struct{ proc *pty.PTYProcess }

func (m ptyMaster) Read(p []byte) (int, error)     { return m.proc.DataConn().Read(p) }
func (m ptyMaster) Write(p []byte) (int, error)    { return m.proc.DataConn().Write(p) }
func (m ptyMaster) Close() error                   { return m.proc.DataConn().Close() }
func (m ptyMaster) Resize(rows, cols uint16) error { return m.proc.SetSize(rows, cols) }

// winState is one daemon-owned window: its PTY master and the data
// connections currently relaying for it.
type winState struct {
	id        int
	master    masterConn
	attachers map[net.Conn]struct{}
}

// state tracks the daemon's windows and serializes session-file updates.
type state struct {
	mu         sync.Mutex
	windows    map[int]*winState
	sessionObj *session.Session
	// shutDown is set once cleanup starts; attachers unregistering after
	// that must not write the session file back (it would resurrect the
	// file cleanup just removed).
	shutDown bool
	// endSession tears the whole session down (files, processes); set by
	// Run, invoked when the last window's program exits.
	endSession func()
}

// updateSession persists the daemon-maintained runtime fields (attach
// state derived from live connections plus any window mutations).
func (st *state) updateSession() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.saveLocked()
}

// saveLocked persists the session file; caller holds st.mu.
func (st *state) saveLocked() {
	if st.shutDown || st.sessionObj == nil {
		return
	}
	attached := false
	for _, w := range st.windows {
		if len(w.attachers) > 0 {
			attached = true
			break
		}
	}
	st.sessionObj.Attached = attached
	_ = st.sessionObj.Save()
}

// Run serves a session until its last window exits or QUIT is received.
// initial may be nil (Windows bootstrap): the daemon then starts the
// initial program from SGREEN_SESSION_* environment. Run never returns
// normally except on listen failure; all other exits go through cleanup +
// os.Exit.
func Run(initial *pty.PTYProcess, sessionFile, sessionID, socketPath string, sessionPid int, ready *os.File) {
	logger := log.New(log.Writer(), "sgreen-daemon: ", log.LstdFlags)

	// Load the session file the creating process wrote, then take ownership
	// of the runtime fields.
	sess, err := session.LoadFromFile(sessionFile, sessionID)
	if err != nil {
		logger.Printf("failed to load session file %s: %v", sessionFile, err)
		sess = &session.Session{ID: sessionID}
	} else {
		logger.Printf("loaded session %q daemon_pid=%d attached=%v windows=%d",
			sess.ID, sess.DaemonPid, sess.Attached, len(sess.Windows))
	}

	st := &state{
		windows:    map[int]*winState{},
		sessionObj: sess,
		endSession: func() {},
	}

	// Bootstrap mode: no PTY was handed over, so the daemon starts the
	// initial program itself from the environment.
	if initial == nil {
		env := map[string]string{"TERM": "screen"}
		if t := os.Getenv("SGREEN_SESSION_TERM"); t != "" && t != "-" {
			env["TERM"] = t
		}
		proc, err := pty.StartWithEnv(
			os.Getenv("SGREEN_SESSION_CMD"),
			envStringSlice(os.Getenv("SGREEN_SESSION_ARGS")),
			env)
		if err != nil {
			logger.Printf("bootstrap window: %v", err)
			signalReady(ready)
			return
		}
		initial = proc
		if win := sess.GetCurrentWindow(); win != nil {
			win.Pid = proc.Cmd.Process.Pid
		}
		sess.Pid = proc.Cmd.Process.Pid
	}

	initialWin := &winState{
		id:        sess.CurrentWindow,
		master:    ptyMaster{proc: initial},
		attachers: make(map[net.Conn]struct{}),
	}
	st.windows[initialWin.id] = initialWin

	ln, authSecret, err := listenTransport(socketPath)
	if err != nil {
		logger.Printf("failed to listen on %s: %v", socketPath, err)
		// The parent is waiting on the ready pipe; unblock it so it can
		// fall back instead of hanging.
		signalReady(ready)
		return
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			logger.Printf("cleanup: starting")
			st.mu.Lock()
			st.shutDown = true
			for _, w := range st.windows {
				for conn := range w.attachers {
					_ = conn.Close()
				}
				_ = w.master.Close()
				killPid(windowPidLocked(st, w.id))
			}
			st.windows = map[int]*winState{}
			st.mu.Unlock()
			_ = ln.Close()
			_ = os.Remove(socketPath)
			if err := session.RemoveFile(sessionFile, sessionID); err != nil {
				logger.Printf("cleanup: remove session file: %v", err)
			}
			if sessionPid > 0 {
				killPid(sessionPid)
			}
			logger.Printf("cleanup: done")
		})
	}

	// Exit paths: last window exited, QUIT command, signal.
	sigChan := make(chan os.Signal, 1)
	notifySignals(sigChan)
	go func() {
		for range sigChan {
			cleanup()
			exitFunc(0)
		}
	}()

	// Per-window relay: fan PTY output out to attached clients; ends when
	// the master is closed or errors (window program exited).
	st.endSession = func() {
		cleanup()
		exitFunc(0)
	}
	startWindowRelay(st, initialWin, logger)

	// Take ownership of the runtime fields, then unblock the parent.
	st.mu.Lock()
	st.sessionObj.DaemonPid = os.Getpid()
	st.saveLocked()
	st.mu.Unlock()
	signalReady(ready)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// The listener was closed by a cleanup path (QUIT, last window,
			// signal). That path finishes cleanup and calls os.Exit itself;
			// returning here would race main()'s exit and could kill the
			// process before cleanup removed the session files.
			select {}
		}
		go handleConn(authenticate(conn, authSecret), st, logger, cleanup)
	}
}

// authenticate validates the AUTH <token> handshake line required by
// transports that are reachable beyond filesystem permissions (Windows
// loopback TCP). Unix sockets skip it (empty secret).
func authenticate(conn net.Conn, secret string) net.Conn {
	if secret == "" {
		return conn
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, err := readLine(conn)
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil || strings.TrimSpace(line) != "AUTH "+secret {
		_, _ = conn.Write([]byte("ERR auth\n"))
		_ = conn.Close()
		return deadConn{}
	}
	return conn
}

// deadConn swallows I/O on a rejected connection.
type deadConn struct{}

func (deadConn) Read([]byte) (int, error)         { return 0, io.EOF }
func (deadConn) Write(p []byte) (int, error)      { return len(p), nil }
func (deadConn) Close() error                     { return nil }
func (deadConn) SetDeadline(time.Time) error      { return nil }
func (deadConn) SetReadDeadline(time.Time) error  { return nil }
func (deadConn) SetWriteDeadline(time.Time) error { return nil }
func (deadConn) LocalAddr() net.Addr              { return nil }
func (deadConn) RemoteAddr() net.Addr             { return nil }

// startWindowRelay spawns the relay goroutine for a window; when the
// window's program exits (master read ends) the window is removed, and the
// session ends once the last window is gone.
func startWindowRelay(st *state, w *winState, logger *log.Logger) {
	go relayWindow(st, w, logger, func() {
		// Resolve the window by identity: IDs are renumbered when other
		// windows die, so w.id may be stale - or already reused by a
		// different window - by the time this fires.
		st.mu.Lock()
		id := -1
		for i, cur := range st.windows {
			if cur == w {
				id = i
				break
			}
		}
		st.mu.Unlock()
		if id >= 0 {
			removeWindow(st, id, logger)
		}
		if len(st.snapshotWindows()) == 0 && st.endSession != nil {
			st.endSession()
		}
	})
}

func signalReady(ready *os.File) {
	if ready == nil {
		return
	}
	_, _ = ready.Write([]byte("ready\n"))
	_ = ready.Close()
}

// snapshotWindows returns the current window states under lock.
func (st *state) snapshotWindows() []*winState {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]*winState, 0, len(st.windows))
	for _, w := range st.windows {
		out = append(out, w)
	}
	return out
}

// windowPidLocked resolves a window's program pid; caller holds st.mu.
func windowPidLocked(st *state, winID int) int {
	if st.sessionObj == nil {
		return 0
	}
	for _, sw := range st.sessionObj.Windows {
		if sw != nil && sw.ID == winID {
			return sw.Pid
		}
	}
	return 0
}

// relayWindow fans one window's PTY output out to its attached clients.
func relayWindow(st *state, w *winState, logger *log.Logger, onEnd func()) {
	buf := make([]byte, 32*1024)
	for {
		n, err := w.master.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			st.mu.Lock()
			for conn := range w.attachers {
				if _, werr := conn.Write(data); werr != nil {
					_ = conn.Close()
					delete(w.attachers, conn)
				}
			}
			st.mu.Unlock()
		}
		if err != nil {
			logger.Printf("window %d read end: %v", w.id, err)
			onEnd()
			return
		}
	}
}

// removeWindow drops a window (program exited or killed): unregister it,
// close its attachers, splice it out of the session file, and renumber.
// Session end (last window) is handled by the callers.
func removeWindow(st *state, winID int, logger *log.Logger) {
	st.mu.Lock()
	w, ok := st.windows[winID]
	if !ok {
		st.mu.Unlock()
		return
	}
	delete(st.windows, winID)
	for conn := range w.attachers {
		_ = conn.Close()
	}
	w.attachers = map[net.Conn]struct{}{}

	if st.sessionObj != nil && !st.shutDown && len(st.sessionObj.Windows) > 0 {
		wasCurrent := st.sessionObj.CurrentWindow == winID
		idx := -1
		for i, sw := range st.sessionObj.Windows {
			if sw != nil && sw.ID == winID {
				idx = i
				break
			}
		}
		if idx >= 0 {
			st.sessionObj.Windows = append(st.sessionObj.Windows[:idx], st.sessionObj.Windows[idx+1:]...)
			for i, sw := range st.sessionObj.Windows {
				sw.ID = i
				sw.Number = session.WindowNumberString(i)
			}
			if len(st.sessionObj.Windows) == 0 {
				st.sessionObj.CurrentWindow = 0
			} else if wasCurrent || st.sessionObj.CurrentWindow >= len(st.sessionObj.Windows) {
				st.sessionObj.CurrentWindow = 0
			}
			if st.sessionObj.LastWindow >= len(st.sessionObj.Windows) {
				st.sessionObj.LastWindow = 0
			}
			// Rekey the daemon window states to match the renumbered
			// session file: both represent the same surviving windows in
			// the same order, so ascending old IDs map positionally onto
			// the new 0..n-1 IDs. Without this, lookups by window ID
			// (ATTACH, RESIZE, STUFF, KILLWINDOW) miss after a kill.
			ids := make([]int, 0, len(st.windows))
			for id := range st.windows {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			if len(ids) == len(st.sessionObj.Windows) {
				rekeyed := make(map[int]*winState, len(ids))
				for newID, oldID := range ids {
					rekeyed[newID] = st.windows[oldID]
				}
				st.windows = rekeyed
			}
			st.saveLocked()
		}
	}
	remaining := len(st.windows)
	st.mu.Unlock()
	logger.Printf("window %d removed; %d windows remain", winID, remaining)
}

// handleConn reads the protocol first line and dispatches to ATTACH relay
// or CONTROL command handling.
func handleConn(conn net.Conn, st *state, logger *log.Logger, onQuit func()) {
	line, err := readLine(conn)
	if err != nil {
		_ = conn.Close()
		return
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		_, _ = conn.Write([]byte("ERR empty mode\n"))
		_ = conn.Close()
		return
	}
	switch parts[0] {
	case "ATTACH":
		serveAttach(conn, st, logger, parts[1:])
	case "CONTROL":
		serveControl(conn, st, logger, onQuit)
	default:
		_, _ = conn.Write([]byte("ERR unknown mode\n"))
		_ = conn.Close()
	}
}

// serveAttach relays one attached client for one window until the
// connection closes.
func serveAttach(conn net.Conn, st *state, logger *log.Logger, args []string) {
	winID := 0
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil {
			winID = v
		}
	}

	st.mu.Lock()
	w, ok := st.windows[winID]
	if !ok {
		st.mu.Unlock()
		_, _ = conn.Write([]byte("ERR no such window\n"))
		_ = conn.Close()
		return
	}
	w.attachers[conn] = struct{}{}
	st.mu.Unlock()
	logger.Printf("attacher registered on window %d", winID)
	st.updateSession()

	defer func() {
		st.mu.Lock()
		delete(w.attachers, conn)
		st.mu.Unlock()
		st.updateSession()
		_ = conn.Close()
	}()

	buf := make([]byte, 32*1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			if _, werr := w.master.Write(buf[:n]); werr != nil {
				logger.Printf("window %d write: %v", winID, werr)
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// serveControl executes control commands, one per line, until EOF.
func serveControl(conn net.Conn, st *state, logger *log.Logger, onQuit func()) {
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
			n := 0
			current := 0
			if st.sessionObj != nil {
				current = st.sessionObj.CurrentWindow
			}
			for _, w := range st.windows {
				n += len(w.attachers)
			}
			st.mu.Unlock()
			if n > 0 {
				_, _ = fmt.Fprintf(conn, "OK ATTACHED %d %d\n", n, current)
			} else {
				_, _ = fmt.Fprintf(conn, "OK DETACHED %d\n", current)
			}
		case "DETACH", "POWER-DETACH":
			kickAttachers(st)
			_, _ = conn.Write([]byte("OK\n"))
		case "RESIZE":
			// RESIZE <winid> <rows> <cols>
			if len(parts) != 4 {
				_, _ = conn.Write([]byte("ERR usage: RESIZE winid rows cols\n"))
				continue
			}
			wid, err1 := strconv.Atoi(parts[1])
			rows, err2 := strconv.Atoi(parts[2])
			cols, err3 := strconv.Atoi(parts[3])
			if err1 != nil || err2 != nil || err3 != nil || rows <= 0 || cols <= 0 {
				_, _ = conn.Write([]byte("ERR bad size\n"))
				continue
			}
			st.mu.Lock()
			w := st.windows[wid]
			st.mu.Unlock()
			if w == nil {
				_, _ = conn.Write([]byte("ERR no such window\n"))
				continue
			}
			if err := w.master.Resize(uint16(rows), uint16(cols)); err != nil {
				_, _ = fmt.Fprintf(conn, "ERR %v\n", err)
				continue
			}
			_, _ = conn.Write([]byte("OK\n"))
		case "STUFF":
			// STUFF <base64data> — writes to the current window's program.
			if len(parts) != 2 {
				_, _ = conn.Write([]byte("ERR usage: STUFF base64data\n"))
				continue
			}
			data, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				_, _ = conn.Write([]byte("ERR bad base64\n"))
				continue
			}
			st.mu.Lock()
			cur := 0
			if st.sessionObj != nil {
				cur = st.sessionObj.CurrentWindow
			}
			w := st.windows[cur]
			st.mu.Unlock()
			if w == nil {
				_, _ = conn.Write([]byte("ERR no current window\n"))
				continue
			}
			if _, err := w.master.Write(data); err != nil {
				_, _ = fmt.Fprintf(conn, "ERR %v\n", err)
				continue
			}
			_, _ = conn.Write([]byte("OK\n"))
		case "NEWWINDOW":
			// NEWWINDOW <base64 cmd> <base64 json-args> <term>
			if len(parts) != 4 {
				_, _ = conn.Write([]byte("ERR usage: NEWWINDOW cmd args term\n"))
				continue
			}
			newID, err := createWindowInDaemon(st, parts[1], parts[2], parts[3], logger)
			if err != nil {
				_, _ = fmt.Fprintf(conn, "ERR %v\n", err)
				continue
			}
			_, _ = fmt.Fprintf(conn, "OK %d\n", newID)
		case "NEXT", "PREV", "TOGGLE", "SELECT":
			arg := ""
			if len(parts) > 1 {
				arg = parts[1]
			}
			newCur, err := switchWindow(st, strings.ToLower(parts[0]), arg)
			if err != nil {
				_, _ = fmt.Fprintf(conn, "ERR %v\n", err)
				continue
			}
			_, _ = fmt.Fprintf(conn, "OK %d\n", newCur)
		case "KILLWINDOW":
			if len(parts) != 2 {
				_, _ = conn.Write([]byte("ERR usage: KILLWINDOW winid\n"))
				continue
			}
			wid, err := strconv.Atoi(parts[1])
			if err != nil {
				_, _ = conn.Write([]byte("ERR bad window id\n"))
				continue
			}
			st.mu.Lock()
			w := st.windows[wid]
			count := len(st.windows)
			st.mu.Unlock()
			if w == nil {
				_, _ = conn.Write([]byte("ERR no such window\n"))
				continue
			}
			killPid(windowPidLocked(st, wid))
			_ = w.master.Close()
			// The relay goroutine observes master EOF; force removal too so
			// the reply reflects the new state.
			removeWindow(st, wid, logger)
			if count <= 1 {
				// GNU semantics: killing the last window terminates the
				// whole session ("[screen is terminating]").
				_, _ = conn.Write([]byte("OK\n"))
				st.endSession()
				return
			}
			_, _ = conn.Write([]byte("OK\n"))
		case "SETTITLE":
			// SETTITLE <winid> <base64 title>
			if len(parts) != 3 {
				_, _ = conn.Write([]byte("ERR usage: SETTITLE winid title\n"))
				continue
			}
			wid, err1 := strconv.Atoi(parts[1])
			title, err2 := base64.StdEncoding.DecodeString(parts[2])
			if err1 != nil || err2 != nil {
				_, _ = conn.Write([]byte("ERR bad arguments\n"))
				continue
			}
			st.mu.Lock()
			found := false
			if st.sessionObj != nil {
				for _, sw := range st.sessionObj.Windows {
					if sw != nil && sw.ID == wid {
						sw.Title = string(title)
						found = true
						break
					}
				}
				if found {
					st.saveLocked()
				}
			}
			st.mu.Unlock()
			if !found {
				_, _ = conn.Write([]byte("ERR no such window\n"))
				continue
			}
			_, _ = conn.Write([]byte("OK\n"))
		case "QUIT":
			_, _ = conn.Write([]byte("OK\n"))
			logger.Printf("QUIT received")
			onQuit()
			// Exit here so the cleanup above (file removal) completes
			// atomically instead of racing main()'s return.
			exitFunc(0)
		default:
			_, _ = fmt.Fprintf(conn, "ERR unknown command %q\n", parts[0])
		}
	}
}

// createWindowInDaemon spawns a new program on a daemon-owned PTY and
// records the window in the session file.
func createWindowInDaemon(st *state, b64Cmd, b64Args, term string, logger *log.Logger) (int, error) {
	cmdPath, err := base64.StdEncoding.DecodeString(b64Cmd)
	if err != nil {
		return 0, fmt.Errorf("bad command encoding")
	}
	argsJSON, err := base64.StdEncoding.DecodeString(b64Args)
	if err != nil {
		return 0, fmt.Errorf("bad args encoding")
	}
	var args []string
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return 0, fmt.Errorf("bad args: %w", err)
		}
	}

	envOverrides := map[string]string{"TERM": "screen"}
	if t := string(term); t != "" && t != "-" {
		envOverrides["TERM"] = t
	}
	ptyProc, err := pty.StartWithEnv(string(cmdPath), args, envOverrides)
	if err != nil {
		return 0, fmt.Errorf("start pty: %w", err)
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.shutDown {
		_ = ptyProc.Kill()
		return 0, fmt.Errorf("daemon is shutting down")
	}

	newID := len(st.sessionObj.Windows)
	w := &winState{
		id:        newID,
		master:    ptyMaster{proc: ptyProc},
		attachers: make(map[net.Conn]struct{}),
	}
	st.windows[newID] = w

	window := &session.Window{
		ID:             newID,
		Number:         session.WindowNumberString(newID),
		CmdPath:        string(cmdPath),
		CmdArgs:        args,
		Pid:            ptyProc.Cmd.Process.Pid,
		PtsPath:        ptyProc.PtsPath,
		CreatedAt:      time.Now(),
		ScrollbackSize: 1000,
		Encoding:       session.DetectLocaleEncoding(),
	}
	st.sessionObj.Windows = append(st.sessionObj.Windows, window)
	st.sessionObj.LastWindow = st.sessionObj.CurrentWindow
	st.sessionObj.CurrentWindow = newID
	st.saveLocked()
	logger.Printf("window %d created (pid %d, cmd %q)", newID, window.Pid, cmdPath)

	startWindowRelay(st, w, logger)
	return newID, nil
}

// switchWindow moves the session cursor and persists it.
func switchWindow(st *state, op string, arg string) (int, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.sessionObj == nil {
		return 0, fmt.Errorf("no session")
	}
	n := len(st.sessionObj.Windows)
	if n == 0 {
		return 0, fmt.Errorf("no windows")
	}

	validate := func(idx int) (int, error) {
		if idx < 0 || idx >= n {
			return 0, fmt.Errorf("window out of range")
		}
		return idx, nil
	}

	var err error
	cur := st.sessionObj.CurrentWindow
	next := cur
	switch op {
	case "next":
		next = (cur + 1) % n
	case "prev":
		next = (cur + n - 1) % n
	case "toggle":
		next = st.sessionObj.LastWindow
	case "select":
		next = windowArgToIndex(st.sessionObj, arg)
	default:
		err = fmt.Errorf("unknown switch op %q", op)
	}
	if err != nil {
		return 0, err
	}
	next, err = validate(next)
	if err != nil {
		return 0, err
	}
	st.sessionObj.LastWindow = cur
	st.sessionObj.CurrentWindow = next
	st.saveLocked()
	return next, nil
}

// windowArgToIndex resolves a window number string ("0".."9", "A".."Z") to
// a window index, mirroring the session package's semantics.
func windowArgToIndex(sess *session.Session, arg string) int {
	for _, w := range sess.Windows {
		if w != nil && w.Number == arg {
			return w.ID
		}
	}
	return -1
}

func kickAttachers(st *state) {
	st.mu.Lock()
	conns := make([]net.Conn, 0)
	for _, w := range st.windows {
		for conn := range w.attachers {
			conns = append(conns, conn)
		}
		w.attachers = map[net.Conn]struct{}{}
	}
	st.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Close()
	}
	st.updateSession()
}

// exitFunc is os.Exit, replaceable in tests so a session ending inside the
// test process does not kill the test binary.
var exitFunc = os.Exit

// envStringSlice decodes a JSON string slice from an environment variable.
func envStringSlice(v string) []string {
	if v == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		return nil
	}
	return out
}
