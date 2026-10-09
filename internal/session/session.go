package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/inoki/sgreen/internal/pty"
)

// Config represents session configuration options
type Config struct {
	Term            string
	UTF8            bool
	Scrollback      int
	AllCapabilities bool
	Encoding        string // Window encoding (e.g., UTF-8, ISO-8859-1)
}

// Session represents a screen session
type Session struct {
	ID           string         `json:"id"`
	CmdPath      string         `json:"cmd_path"`
	CmdArgs      []string       `json:"cmd_args"`
	Pid          int            `json:"pid"`
	PtsPath      string         `json:"pts_path,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	Owner        string         `json:"owner,omitempty"`
	AllowedUsers []string       `json:"allowed_users,omitempty"`
	Layouts      map[string]int `json:"layouts,omitempty"`

	// Runtime state maintained by the session daemon (Unix). Attached is
	// true while at least one client is attached; DaemonPid identifies the
	// relay daemon that owns the PTY master.
	Attached  bool `json:"attached,omitempty"`
	DaemonPid int  `json:"daemon_pid,omitempty"`

	// Window management
	Windows       []*Window `json:"windows,omitempty"`     // All windows in this session
	CurrentWindow int       `json:"current_window"`        // Index of current window
	LastWindow    int       `json:"last_window,omitempty"` // Index of last window (for C-a C-a)

	// Runtime fields (not persisted)
	PTYProcess *pty.PTYProcess `json:"-"` // Deprecated: use Windows[CurrentWindow] instead
	// Controller, when set, routes window mutations to the session daemon
	// that owns the PTY masters (set by the attach path; see controller.go).
	Controller WindowController `json:"-"`
	mu         sync.RWMutex     `json:"-"`
}

var (
	sessions     = make(map[string]*Session)
	sessionsMu   sync.RWMutex
	sessionsDir  string
	sessionsOnce sync.Once
)

// dir returns the on-disk sessions directory, creating it on first use.
// Initialization is lazy so tests can point HOME at a scratch directory
// before touching any session API (the old package-level init() captured
// HOME at process start, which made test isolation impossible).
func dir() string {
	sessionsOnce.Do(func() {
		// GNU screen relocates its sockets via $SCREENDIR; honor it so the
		// directory sgreen actually uses matches what -ls reports.
		if screenDir := os.Getenv("SCREENDIR"); screenDir != "" {
			sessionsDir = screenDir
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = os.TempDir()
			}
			sessionsDir = filepath.Join(homeDir, ".sgreen", "sessions")
		}
		if err := os.MkdirAll(sessionsDir, 0755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: failed to create sessions directory: %v\n", err)
		}
	})
	return sessionsDir
}

// CurrentUser returns the current username for permission checks.
func CurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return ""
}

// New creates a new session with the given ID, command, and arguments
func New(id, cmdPath string, args []string) (*Session, error) {
	return NewWithConfig(id, cmdPath, args, nil)
}

// NewWithConfig creates a new session with configuration options
func NewWithConfig(id, cmdPath string, args []string, config *Config) (*Session, error) {
	// Validate session name
	if id == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}
	for _, r := range id {
		if !isValidSessionChar(r) {
			return nil, fmt.Errorf("invalid session name: only alphanumeric characters, dot, dash, and underscore allowed")
		}
	}

	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	// Check if session already exists
	if _, exists := sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	// Build environment overrides
	envOverrides := make(map[string]string)
	if config != nil {
		// Set TERM if specified, otherwise default to screen
		if config.Term != "" {
			envOverrides["TERM"] = config.Term
		} else {
			// Default to screen, or screen-256color if all capabilities requested
			envOverrides["TERM"] = "screen"
		}

		// Set UTF-8 locale if UTF-8 mode is enabled
		if config.UTF8 {
			if locale := os.Getenv("LANG"); locale != "" {
				// Ensure locale has UTF-8
				if !strings.Contains(locale, "UTF-8") && !strings.Contains(locale, "utf8") {
					parts := strings.Split(locale, ".")
					envOverrides["LANG"] = parts[0] + ".UTF-8"
				}
			} else {
				envOverrides["LANG"] = "en_US.UTF-8"
			}
		}

		// Set SCREENCAP if all capabilities requested
		// This tells screen to include all capabilities even if terminal lacks them
		// For sgreen, we set a flag that can be used later
		if config.AllCapabilities {
			// Set TERM to screen-256color to indicate full capabilities
			if envOverrides["TERM"] == "screen" {
				envOverrides["TERM"] = "screen-256color"
			}
		}
	} else {
		// Default TERM to screen
		envOverrides["TERM"] = "screen"
	}

	// Start PTY process with environment overrides. On Windows this
	// returns nil, nil: no fd inheritance exists there, so the session
	// daemon (forked next) starts the program itself.
	ptyProc, err := startSessionPTY(cmdPath, args, envOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Determine encoding for the window
	encoding := ""
	if config != nil {
		if config.Encoding != "" {
			encoding = config.Encoding
		} else if config.UTF8 {
			encoding = "UTF-8"
		}
	}
	if encoding == "" {
		encoding = detectEncodingFromLocale()
	}

	// Create first window
	scrollbackSize := 1000 // Default
	if config != nil && config.Scrollback > 0 {
		scrollbackSize = config.Scrollback
	}
	windowPid := 0
	windowPts := ""
	if ptyProc != nil && ptyProc.Cmd != nil && ptyProc.Cmd.Process != nil {
		windowPid = ptyProc.Cmd.Process.Pid
		windowPts = ptyProc.PtsPath
	}
	window := &Window{
		ID:             0,
		Number:         "0",
		Title:          "",
		CmdPath:        cmdPath,
		CmdArgs:        args,
		Pid:            windowPid,
		PtsPath:        windowPts,
		CreatedAt:      time.Now(),
		ScrollbackSize: scrollbackSize,
		Encoding:       encoding,
		PTYProcess:     ptyProc,
	}

	// Create session
	sess := &Session{
		ID:            id,
		CmdPath:       cmdPath,
		CmdArgs:       args,
		Pid:           windowPid,
		PtsPath:       windowPts,
		CreatedAt:     time.Now(),
		Owner:         CurrentUser(),
		Windows:       []*Window{window},
		CurrentWindow: 0,
		LastWindow:    0,
		PTYProcess:    ptyProc, // Deprecated: kept for backward compatibility
	}

	// Store in memory
	sessions[id] = sess

	// Persist to disk
	if err := sess.save(); err != nil {
		// Clean up on error
		delete(sessions, id)
		if ptyProc != nil {
			_ = ptyProc.Kill()
		}
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return sess, nil
}

// Load loads a session by ID
func Load(id string) (*Session, error) {
	sessionsMu.RLock()
	// Check in-memory first
	if sess, exists := sessions[id]; exists {
		sessionsMu.RUnlock()
		return sess, nil
	}
	sessionsMu.RUnlock()

	// Load from disk
	sess, err := loadFromDisk(id)
	if err != nil {
		return nil, err
	}

	// NOTE: the PTY master is owned by the session daemon; clients attach
	// through the daemon socket. Opening the slave device here would
	// corrupt the terminal line discipline, so no PTY reconnect happens.

	sessionsMu.Lock()
	sessions[id] = sess
	sessionsMu.Unlock()

	return sess, nil
}

// detectEncodingFromLocale detects encoding from locale environment variables.
func detectEncodingFromLocale() string {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		locale := os.Getenv(key)
		if locale == "" {
			continue
		}
		parts := strings.Split(locale, ".")
		if len(parts) < 2 {
			continue
		}
		encoding := strings.ToUpper(parts[1])
		encoding = strings.ReplaceAll(encoding, "_", "-")
		switch encoding {
		case "UTF-8", "UTF8":
			return "UTF-8"
		case "ISO-8859-1", "ISO8859-1", "LATIN1":
			return "ISO-8859-1"
		case "ISO-8859-2", "ISO8859-2", "LATIN2":
			return "ISO-8859-2"
		case "ISO-8859-15", "ISO8859-15", "LATIN9":
			return "ISO-8859-15"
		case "WINDOWS-1252", "CP1252":
			return "WINDOWS-1252"
		case "WINDOWS-1251", "CP1251":
			return "WINDOWS-1251"
		case "KOI8-R", "KOI8R":
			return "KOI8-R"
		case "KOI8-U", "KOI8U":
			return "KOI8-U"
		}
	}
	return "UTF-8"
}

func isResourceExhausted(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EMFILE) || errors.Is(err, syscall.ENFILE)
}

// List returns all active sessions (from both memory and disk)
func List() []*Session {
	sessionsMu.RLock()
	memorySessions := make(map[string]*Session)
	for id, sess := range sessions {
		memorySessions[id] = sess
	}
	sessionsMu.RUnlock()

	// Load all sessions from disk
	diskSessions, err := loadAllFromDisk()
	if err != nil {
		// If we can't read from disk, just return memory sessions
		result := make([]*Session, 0, len(memorySessions))
		for _, sess := range memorySessions {
			result = append(result, sess)
		}
		return result
	}

	// Merge memory and disk sessions (memory takes precedence)
	result := make([]*Session, 0, len(memorySessions)+len(diskSessions))
	seen := make(map[string]bool)

	// Add memory sessions first
	for _, sess := range memorySessions {
		result = append(result, sess)
		seen[sess.ID] = true
	}

	// Add disk sessions that aren't in memory. The PTY master belongs to
	// the session daemon; cross-process attach goes through the daemon
	// socket, so there is nothing to reconnect here.
	for _, sess := range diskSessions {
		if !seen[sess.ID] {
			result = append(result, sess)
		}
	}

	// Keep dead sessions in the list; -wipe removes them explicitly.
	return result
}

// loadAllFromDisk loads all session files from disk
func loadAllFromDisk() ([]*Session, error) {
	entries, err := os.ReadDir(dir())
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		sess, err := loadFromDisk(id)
		if err != nil {
			// Skip invalid session files
			continue
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// save persists the session to disk
func (s *Session) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(dir(), s.ID+".json")
	// Ensure sessions directory exists
	if err := os.MkdirAll(dir(), 0755); err != nil {
		if isResourceExhausted(err) {
			return fmt.Errorf("resource exhaustion while creating sessions directory: %w", err)
		}
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		if isResourceExhausted(err) {
			return fmt.Errorf("resource exhaustion while writing session file: %w", err)
		}
		return fmt.Errorf("failed to write session file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		if isResourceExhausted(err) {
			return fmt.Errorf("resource exhaustion while renaming session file: %w", err)
		}
		return fmt.Errorf("failed to rename session file: %w", err)
	}

	return nil
}

// loadFromDisk loads a session from disk
func loadFromDisk(id string) (*Session, error) {
	filePath := filepath.Join(dir(), id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %s not found", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		// Try to recover by backing up corrupted file
		backupPath := filePath + ".corrupted"
		_ = os.WriteFile(backupPath, data, 0644)
		return nil, fmt.Errorf("failed to parse session file (backed up to %s): %w", backupPath, err)
	}

	// Validate session structure
	if sess.ID == "" {
		return nil, fmt.Errorf("invalid session: missing ID")
	}
	if sess.ID != id {
		// ID mismatch, fix it
		sess.ID = id
	}
	if sess.Owner == "" {
		sess.Owner = CurrentUser()
	}

	return &sess, nil
}

// SocketPath returns the daemon socket path for a session ID.
func SocketPath(id string) string {
	return filepath.Join(dir(), id+".sock")
}

// FilePath returns the JSON session file path for a session ID.
func FilePath(id string) string {
	return filepath.Join(dir(), id+".json")
}

// LoadFromFile loads a session from an explicit file path without
// consulting or populating the in-memory registry. Used by the session
// daemon, which owns the runtime fields of its session file.
func LoadFromFile(path, id string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	if sess.ID == "" {
		sess.ID = id
	}
	return &sess, nil
}

// RemoveFile deletes the session file and socket for a session that is
// being torn down by its daemon.
func RemoveFile(path, id string) error {
	_ = os.Remove(SocketPath(id))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	sessionsMu.Lock()
	delete(sessions, id)
	sessionsMu.Unlock()
	return nil
}

// Delete removes a session from memory and disk. For sessions owned by
// another process (disk-only), this removes the files without touching the
// owning processes; use the daemon QUIT command to terminate a live
// session cleanly.
func Delete(id string) error {
	sessionsMu.Lock()
	sess, exists := sessions[id]
	if exists {
		delete(sessions, id)
	}
	sessionsMu.Unlock()

	if exists {
		// Locally-owned session: kill all processes in all windows.
		for _, win := range sess.Windows {
			if win.GetPTYProcess() != nil {
				_ = win.GetPTYProcess().Kill()
			}
		}
		// Also kill legacy PTY process if exists.
		if sess.PTYProcess != nil {
			_ = sess.PTYProcess.Kill()
		}
	}

	// Remove from disk (session file and daemon socket).
	if err := os.Remove(filepath.Join(dir(), id+".json")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session file: %w", err)
	}
	_ = os.Remove(SocketPath(id))

	return nil
}

// GetPTYProcess returns the PTY process for this session
// Deprecated: Use GetCurrentWindow().GetPTYProcess() instead
func (s *Session) GetPTYProcess() *pty.PTYProcess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Try to get from current window first
	if len(s.Windows) > 0 && s.CurrentWindow < len(s.Windows) {
		if win := s.Windows[s.CurrentWindow]; win != nil {
			if ptyProc := win.GetPTYProcess(); ptyProc != nil {
				return ptyProc
			}
		}
	}
	// Fallback to deprecated field
	return s.PTYProcess
}

// GetCurrentWindow returns the current window
func (s *Session) GetCurrentWindow() *Window {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.Windows) == 0 || s.CurrentWindow < 0 || s.CurrentWindow >= len(s.Windows) {
		return nil
	}
	return s.Windows[s.CurrentWindow]
}

// GetWindow returns a window by its number (0-9, A-Z)
func (s *Session) GetWindow(number string) *Window {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, err := windowStringToNumber(number)
	if err != nil {
		return nil
	}
	for _, win := range s.Windows {
		if win.ID == id {
			return win
		}
	}
	return nil
}

// CreateWindow creates a new window in the session
func (s *Session) CreateWindow(cmdPath string, args []string, config *Config) (*Window, error) {
	if s.Controller != nil {
		term := ""
		if config != nil {
			term = config.Term
		}
		if _, err := s.Controller.CreateWindow(cmdPath, args, term); err != nil {
			return nil, err
		}
		if err := s.RefreshFromDisk(); err != nil {
			return nil, err
		}
		return s.GetCurrentWindow(), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find next available window number
	nextID := len(s.Windows)
	if nextID >= 36 {
		return nil, fmt.Errorf("maximum number of windows (36) reached")
	}

	// Build environment overrides
	envOverrides := make(map[string]string)
	if config != nil {
		if config.Term != "" {
			envOverrides["TERM"] = config.Term
		} else {
			envOverrides["TERM"] = "screen"
		}
		if config.AllCapabilities {
			if envOverrides["TERM"] == "screen" {
				envOverrides["TERM"] = "screen-256color"
			}
		}
	} else {
		envOverrides["TERM"] = "screen"
	}

	// Start PTY process
	ptyProc, err := pty.StartWithEnv(cmdPath, args, envOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Determine encoding for the window
	encoding := ""
	if config != nil {
		if config.Encoding != "" {
			encoding = config.Encoding
		} else if config.UTF8 {
			encoding = "UTF-8"
		}
	}
	if encoding == "" {
		encoding = detectEncodingFromLocale()
	}

	// Create window
	scrollbackSize := 1000 // Default
	if config != nil && config.Scrollback > 0 {
		scrollbackSize = config.Scrollback
	}
	window := &Window{
		ID:             nextID,
		Number:         windowNumberToString(nextID),
		Title:          "",
		CmdPath:        cmdPath,
		CmdArgs:        args,
		Pid:            ptyProc.Cmd.Process.Pid,
		PtsPath:        ptyProc.PtsPath,
		CreatedAt:      time.Now(),
		ScrollbackSize: scrollbackSize,
		Encoding:       encoding,
		PTYProcess:     ptyProc,
	}

	// Add to session
	s.Windows = append(s.Windows, window)
	s.LastWindow = s.CurrentWindow
	s.CurrentWindow = nextID

	return window, nil
}

// SwitchToWindow switches to a window by number
func (s *Session) SwitchToWindow(number string) error {
	if s.Controller != nil {
		if _, err := s.Controller.SwitchWindow("select", number); err != nil {
			return err
		}
		return s.RefreshFromDisk()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := windowStringToNumber(number)
	if err != nil {
		return err
	}

	// Find window
	foundIdx := -1
	for i, win := range s.Windows {
		if win.ID == id {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return fmt.Errorf("window %s not found", number)
	}

	s.LastWindow = s.CurrentWindow
	s.CurrentWindow = foundIdx
	return nil
}

// NextWindow switches to the next window
func (s *Session) NextWindow() {
	if s.Controller != nil {
		_, _ = s.Controller.SwitchWindow("next", "")
		_ = s.RefreshFromDisk()
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Windows) == 0 {
		return
	}
	s.LastWindow = s.CurrentWindow
	s.CurrentWindow = (s.CurrentWindow + 1) % len(s.Windows)
}

// PrevWindow switches to the previous window
func (s *Session) PrevWindow() {
	if s.Controller != nil {
		_, _ = s.Controller.SwitchWindow("prev", "")
		_ = s.RefreshFromDisk()
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Windows) == 0 {
		return
	}
	s.LastWindow = s.CurrentWindow
	s.CurrentWindow = (s.CurrentWindow + len(s.Windows) - 1) % len(s.Windows)
}

// ToggleLastWindow switches to the last window
func (s *Session) ToggleLastWindow() {
	if s.Controller != nil {
		_, _ = s.Controller.SwitchWindow("toggle", "")
		_ = s.RefreshFromDisk()
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Windows) == 0 {
		return
	}
	s.CurrentWindow, s.LastWindow = s.LastWindow, s.CurrentWindow
}

// KillCurrentWindow kills the current window
func (s *Session) KillCurrentWindow() error {
	if s.Controller != nil {
		win := s.GetCurrentWindow()
		if win == nil {
			return fmt.Errorf("no current window")
		}
		if err := s.Controller.KillWindow(win.ID); err != nil {
			return err
		}
		return s.RefreshFromDisk()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Windows) == 0 || s.CurrentWindow < 0 || s.CurrentWindow >= len(s.Windows) {
		return fmt.Errorf("no current window")
	}

	// Don't allow killing the last window
	if len(s.Windows) == 1 {
		return fmt.Errorf("cannot kill the last window")
	}

	win := s.Windows[s.CurrentWindow]
	if err := win.Kill(); err != nil {
		return err
	}

	// Remove window from list
	s.Windows = append(s.Windows[:s.CurrentWindow], s.Windows[s.CurrentWindow+1:]...)

	// Renumber windows
	for i, w := range s.Windows {
		w.ID = i
		w.Number = windowNumberToString(i)
	}

	// Adjust current window index
	if s.CurrentWindow >= len(s.Windows) {
		s.CurrentWindow = len(s.Windows) - 1
	}
	if s.LastWindow >= len(s.Windows) {
		s.LastWindow = len(s.Windows) - 1
	}

	return nil
}

// SetWindowTitle sets the title of the current window
func (s *Session) SetWindowTitle(title string) {
	if s.Controller != nil {
		if win := s.GetCurrentWindow(); win != nil {
			win.Title = title // immediate local echo for the UI
			_ = s.Controller.SetTitle(win.ID, title)
		}
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Windows) > 0 && s.CurrentWindow < len(s.Windows) {
		if win := s.Windows[s.CurrentWindow]; win != nil {
			win.Title = title
		}
	}
}

// Rename renames the session
func (s *Session) Rename(newID string) error {
	if newID == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	// Validate session name (alphanumeric, dash, underscore)
	for _, r := range newID {
		if !isValidSessionChar(r) {
			return fmt.Errorf("invalid session name: only alphanumeric characters, dot, dash, and underscore allowed")
		}
	}

	s.mu.Lock()

	// Check if new name already exists
	sessionsMu.RLock()
	if _, exists := sessions[newID]; exists {
		sessionsMu.RUnlock()
		s.mu.Unlock()
		return fmt.Errorf("session %s already exists", newID)
	}
	sessionsMu.RUnlock()

	oldID := s.ID
	oldPath := filepath.Join(dir(), oldID+".json")
	newPath := filepath.Join(dir(), newID+".json")

	// Update in-memory map
	sessionsMu.Lock()
	delete(sessions, oldID)
	s.ID = newID
	sessions[newID] = s
	sessionsMu.Unlock()
	s.mu.Unlock()

	// Rename file on disk
	if err := os.Rename(oldPath, newPath); err != nil {
		// Rollback in-memory change
		s.mu.Lock()
		sessionsMu.Lock()
		delete(sessions, newID)
		s.ID = oldID
		sessions[oldID] = s
		sessionsMu.Unlock()
		s.mu.Unlock()
		return fmt.Errorf("failed to rename session file: %w", err)
	}

	// Save updated session
	return s.save()
}

// Save persists session to disk.
func (s *Session) Save() error {
	return s.save()
}

// CanAttach checks if a user is allowed to attach to this session.
func (s *Session) CanAttach(username string) bool {
	if username == "" {
		return false
	}
	// If no permissions set, allow all (backward compat).
	if len(s.AllowedUsers) == 0 && s.Owner == "" {
		return true
	}
	if s.Owner != "" && s.Owner == username {
		return true
	}
	for _, u := range s.AllowedUsers {
		if u == username {
			return true
		}
	}
	return false
}

// AddUser adds a user to the allowed list.
func (s *Session) AddUser(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	for _, u := range s.AllowedUsers {
		if u == username {
			return nil
		}
	}
	s.AllowedUsers = append(s.AllowedUsers, username)
	return s.save()
}

// RemoveUser removes a user from the allowed list.
func (s *Session) RemoveUser(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	updated := make([]string, 0, len(s.AllowedUsers))
	for _, u := range s.AllowedUsers {
		if u != username {
			updated = append(updated, u)
		}
	}
	s.AllowedUsers = updated
	return s.save()
}

// SaveLayout stores the current window index under a layout name.
func (s *Session) SaveLayout(name string) error {
	if name == "" {
		return fmt.Errorf("layout name cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Layouts == nil {
		s.Layouts = make(map[string]int)
	}
	s.Layouts[name] = s.CurrentWindow
	return s.save()
}

// SelectLayout switches to the window saved under a layout name.
func (s *Session) SelectLayout(name string) error {
	if name == "" {
		return fmt.Errorf("layout name cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Layouts == nil {
		return fmt.Errorf("no layouts available")
	}
	idx, ok := s.Layouts[name]
	if !ok {
		return fmt.Errorf("layout %s not found", name)
	}
	if idx < 0 || idx >= len(s.Windows) {
		return fmt.Errorf("layout %s references invalid window", name)
	}
	s.LastWindow = s.CurrentWindow
	s.CurrentWindow = idx
	return s.save()
}

// ListLayouts returns layout names.
func (s *Session) ListLayouts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.Layouts) == 0 {
		return nil
	}
	names := make([]string, 0, len(s.Layouts))
	for name := range s.Layouts {
		names = append(names, name)
	}
	return names
}

// ExecuteCommand executes a command in a session
func ExecuteCommand(sess *Session, command string) error {
	// Parse command and execute it
	// For now, support basic commands like "quit", "detach", etc.
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := parts[0]

	switch cmd {
	case "quit", "exit":
		// Quit the session
		if sess.PTYProcess != nil {
			_ = sess.PTYProcess.Kill()
		}
		return Delete(sess.ID)
	case "detach":
		// Detach (already handled by Ctrl+A, d)
		return nil
	case "log":
		// Toggle logging (would need to implement)
		return nil
	default:
		// Unknown command
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func isValidSessionChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == '.'
}
