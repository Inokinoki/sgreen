package session

import (
	"encoding/json"
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
	Term          string
	UTF8          bool
	Scrollback    int
	AllCapabilities bool
}

// Session represents a screen session
type Session struct {
	ID        string    `json:"id"`
	CmdPath   string    `json:"cmd_path"`
	CmdArgs   []string  `json:"cmd_args"`
	Pid       int       `json:"pid"`
	PtsPath   string    `json:"pts_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Window management
	Windows     []*Window `json:"windows,omitempty"`     // All windows in this session
	CurrentWindow int     `json:"current_window"`        // Index of current window
	LastWindow    int     `json:"last_window,omitempty"` // Index of last window (for C-a C-a)

	// Runtime fields (not persisted)
	PTYProcess *pty.PTYProcess `json:"-"` // Deprecated: use Windows[CurrentWindow] instead
	mu         sync.RWMutex    `json:"-"`
}

var (
	sessionsDir string
	sessions    = make(map[string]*Session)
	sessionsMu  sync.RWMutex
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	sessionsDir = filepath.Join(homeDir, ".sgreen", "sessions")
	os.MkdirAll(sessionsDir, 0755)
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
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return nil, fmt.Errorf("invalid session name: only alphanumeric characters, dash, and underscore allowed")
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

	// Start PTY process with environment overrides
	ptyProc, err := pty.StartWithEnv(cmdPath, args, envOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Create first window
	scrollbackSize := 1000 // Default
	if config != nil && config.Scrollback > 0 {
		scrollbackSize = config.Scrollback
	}
	window := &Window{
		ID:            0,
		Number:        "0",
		Title:         "",
		CmdPath:       cmdPath,
		CmdArgs:       args,
		Pid:           ptyProc.Cmd.Process.Pid,
		PtsPath:       ptyProc.PtsPath,
		CreatedAt:     time.Now(),
		ScrollbackSize: scrollbackSize,
		PTYProcess:    ptyProc,
	}

	// Create session
	sess := &Session{
		ID:            id,
		CmdPath:       cmdPath,
		CmdArgs:       args,
		Pid:           ptyProc.Cmd.Process.Pid,
		PtsPath:       ptyProc.PtsPath, // Store PTY path for reconnection (backward compat)
		CreatedAt:     time.Now(),
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
		ptyProc.Kill()
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
		// Check if the process is still alive
		if sess.PTYProcess != nil && !sess.PTYProcess.IsAlive() {
			// Process died, try to reconnect if we have a pts path
			if sess.PtsPath != "" {
				sessionsMu.RUnlock()
				sessionsMu.Lock()
				if err := sess.ReconnectPTY(); err == nil {
					sessionsMu.Unlock()
					return sess, nil
				}
				sessionsMu.Unlock()
			}
		}
		sessionsMu.RUnlock()
		return sess, nil
	}
	sessionsMu.RUnlock()

	// Load from disk
	sess, err := loadFromDisk(id)
	if err != nil {
		return nil, err
	}

	// Try to reconnect to the PTY if we have a path and process is still alive
	if sess.PtsPath != "" && isProcessAlive(sess.Pid) {
		if err := sess.ReconnectPTY(); err != nil {
			// Reconnection failed, but continue with session metadata
		}
	}

	sessionsMu.Lock()
	sessions[id] = sess
	sessionsMu.Unlock()

	return sess, nil
}

// ReconnectPTY attempts to reconnect to an existing PTY
func (s *Session) ReconnectPTY() error {
	if s.PtsPath == "" {
		return fmt.Errorf("no PTY path available")
	}

	ptyProc, err := pty.Reconnect(s.PtsPath)
	if err != nil {
		return fmt.Errorf("failed to reconnect to PTY: %w", err)
	}

	s.mu.Lock()
	s.PTYProcess = ptyProc
	s.mu.Unlock()

	return nil
}

// isProcessAlive checks if a process with the given PID is still running
func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// Send signal 0 to check if process exists
	// This doesn't actually send a signal, just checks if the process exists
	err = process.Signal(os.Signal(syscall.Signal(0)))
	return err == nil
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
		// Clean up dead sessions
		if sess.PTYProcess != nil && !sess.PTYProcess.IsAlive() {
			// Try to reconnect if we have a pts path
			if sess.PtsPath != "" && isProcessAlive(sess.Pid) {
				sess.ReconnectPTY()
			}
		}
		result = append(result, sess)
		seen[sess.ID] = true
	}

	// Add disk sessions that aren't in memory
	for _, sess := range diskSessions {
		if !seen[sess.ID] {
			// Try to reconnect windows if processes are still alive
			hasAliveWindow := false
			for _, win := range sess.Windows {
				if win.PtsPath != "" && isProcessAlive(win.Pid) {
					if err := sess.ReconnectPTY(); err == nil {
						hasAliveWindow = true
					}
				}
			}
			// Also try old method for backward compatibility
			if !hasAliveWindow && sess.PtsPath != "" && isProcessAlive(sess.Pid) {
				sess.ReconnectPTY()
			}
			result = append(result, sess)
		}
	}

	// Clean up dead sessions from result
	cleanedResult := make([]*Session, 0, len(result))
	for _, sess := range result {
		hasAliveWindow := false
		if len(sess.Windows) > 0 {
			for _, win := range sess.Windows {
				if win.GetPTYProcess() != nil && win.GetPTYProcess().IsAlive() {
					hasAliveWindow = true
					break
				}
			}
		} else {
			// Fallback check
			if sess.GetPTYProcess() != nil && sess.GetPTYProcess().IsAlive() {
				hasAliveWindow = true
			}
		}
		if hasAliveWindow || len(sess.Windows) == 0 {
			// Keep session if it has alive windows or is a legacy session
			cleanedResult = append(cleanedResult, sess)
		}
	}

	return cleanedResult
}

// loadAllFromDisk loads all session files from disk
func loadAllFromDisk() ([]*Session, error) {
	entries, err := os.ReadDir(sessionsDir)
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

	filePath := filepath.Join(sessionsDir, s.ID+".json")
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// loadFromDisk loads a session from disk
func loadFromDisk(id string) (*Session, error) {
	filePath := filepath.Join(sessionsDir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("session %s not found", id)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &sess, nil
}

// Delete removes a session from memory and disk
func Delete(id string) error {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	sess, exists := sessions[id]
	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	// Kill the process if it's still running
	if sess.PTYProcess != nil {
		sess.PTYProcess.Kill()
	}

	// Remove from memory
	delete(sessions, id)

	// Remove from disk
	filePath := filepath.Join(sessionsDir, id+".json")
	os.Remove(filePath)

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

	// Create window
	scrollbackSize := 1000 // Default
	if config != nil && config.Scrollback > 0 {
		scrollbackSize = config.Scrollback
	}
	window := &Window{
		ID:            nextID,
		Number:        windowNumberToString(nextID),
		Title:         "",
		CmdPath:       cmdPath,
		CmdArgs:       args,
		Pid:           ptyProc.Cmd.Process.Pid,
		PtsPath:       ptyProc.PtsPath,
		CreatedAt:     time.Now(),
		ScrollbackSize: scrollbackSize,
		PTYProcess:    ptyProc,
	}

	// Add to session
	s.Windows = append(s.Windows, window)
	s.LastWindow = s.CurrentWindow
	s.CurrentWindow = nextID

	return window, nil
}

// SwitchToWindow switches to a window by number
func (s *Session) SwitchToWindow(number string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	id, err := windowStringToNumber(number)
	if err != nil {
		return err
	}

	// Find window
	var foundIdx int = -1
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Windows) == 0 {
		return
	}
	s.CurrentWindow, s.LastWindow = s.LastWindow, s.CurrentWindow
}

// KillCurrentWindow kills the current window
func (s *Session) KillCurrentWindow() error {
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
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("invalid session name: only alphanumeric characters, dash, and underscore allowed")
		}
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if new name already exists
	sessionsMu.RLock()
	if _, exists := sessions[newID]; exists {
		sessionsMu.RUnlock()
		return fmt.Errorf("session %s already exists", newID)
	}
	sessionsMu.RUnlock()
	
	oldID := s.ID
	oldPath := filepath.Join(sessionsDir, oldID+".json")
	newPath := filepath.Join(sessionsDir, newID+".json")
	
	// Update in-memory map
	sessionsMu.Lock()
	delete(sessions, oldID)
	s.ID = newID
	sessions[newID] = s
	sessionsMu.Unlock()
	
	// Rename file on disk
	if err := os.Rename(oldPath, newPath); err != nil {
		// Rollback in-memory change
		sessionsMu.Lock()
		delete(sessions, newID)
		s.ID = oldID
		sessions[oldID] = s
		sessionsMu.Unlock()
		return fmt.Errorf("failed to rename session file: %w", err)
	}
	
	// Save updated session
	return s.save()
}

// ForceDetach forces a detach by clearing the PTY process reference
// This allows the session to be reattached from another terminal
func (s *Session) ForceDetach() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Clear the PTY process reference but keep the session alive
	// The process continues running, we just lose the reference
	s.PTYProcess = nil
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
			sess.PTYProcess.Kill()
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

