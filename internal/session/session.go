package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/inoki/sgreen/internal/pty"
)

// Session represents a screen session
type Session struct {
	ID        string    `json:"id"`
	CmdPath   string    `json:"cmd_path"`
	CmdArgs   []string  `json:"cmd_args"`
	Pid       int       `json:"pid"`
	PtsPath   string    `json:"pts_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Runtime fields (not persisted)
	PTYProcess *pty.PTYProcess `json:"-"`
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
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	// Check if session already exists
	if _, exists := sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	// Start PTY process
	ptyProc, err := pty.Start(cmdPath, args)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Create session
	sess := &Session{
		ID:         id,
		CmdPath:    cmdPath,
		CmdArgs:    args,
		Pid:        ptyProc.Cmd.Process.Pid,
		CreatedAt:  time.Now(),
		PTYProcess: ptyProc,
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
		return sess, nil
	}
	sessionsMu.RUnlock()

	// Load from disk
	sess, err := loadFromDisk(id)
	if err != nil {
		return nil, err
	}

	// Try to attach to existing process (if still running)
	// For simplicity, we'll just return the session metadata
	// In a full implementation, you'd check if the process is still alive
	// and potentially reattach to it

	sessionsMu.Lock()
	sessions[id] = sess
	sessionsMu.Unlock()

	return sess, nil
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

	// Add disk sessions that aren't in memory
	for _, sess := range diskSessions {
		if !seen[sess.ID] {
			result = append(result, sess)
		}
	}

	return result
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
func (s *Session) GetPTYProcess() *pty.PTYProcess {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PTYProcess
}

