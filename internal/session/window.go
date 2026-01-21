package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/inoki/sgreen/internal/pty"
)

// Window represents a window within a session
type Window struct {
	ID        int       `json:"id"`        // Window number (0-9, then 10-35 for A-Z)
	Number    string    `json:"number"`    // Display number (0-9, A-Z)
	Title     string    `json:"title"`    // Window title
	CmdPath   string    `json:"cmd_path"` // Command path
	CmdArgs   []string  `json:"cmd_args"` // Command arguments
	Pid       int       `json:"pid"`       // Process ID
	PtsPath   string    `json:"pts_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ScrollbackSize int  `json:"scrollback_size,omitempty"` // Scrollback buffer size

	// Runtime fields (not persisted)
	PTYProcess *pty.PTYProcess `json:"-"`
	mu         sync.RWMutex    `json:"-"`
}

// GetPTYProcess returns the PTY process for this window
func (w *Window) GetPTYProcess() *pty.PTYProcess {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.PTYProcess
}

// SetPTYProcess sets the PTY process for this window
func (w *Window) SetPTYProcess(ptyProc *pty.PTYProcess) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.PTYProcess = ptyProc
	if ptyProc != nil && ptyProc.Cmd != nil && ptyProc.Cmd.Process != nil {
		w.Pid = ptyProc.Cmd.Process.Pid
		w.PtsPath = ptyProc.PtsPath
	}
}

// Kill kills the window's process
func (w *Window) Kill() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.PTYProcess != nil {
		return w.PTYProcess.Kill()
	}
	return nil
}

// IsAlive checks if the window's process is alive
func (w *Window) IsAlive() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.PTYProcess == nil {
		return false
	}
	return w.PTYProcess.IsAlive()
}

// windowNumberToString converts a window ID (0-35) to display string (0-9, A-Z)
func windowNumberToString(id int) string {
	if id < 10 {
		return fmt.Sprintf("%d", id)
	}
	return string(rune('A' + (id - 10)))
}

// windowStringToNumber converts a display string (0-9, A-Z) to window ID
func windowStringToNumber(s string) (int, error) {
	if len(s) == 0 {
		return -1, fmt.Errorf("empty window number")
	}
	
	// Single character
	if len(s) == 1 {
		c := s[0]
		if c >= '0' && c <= '9' {
			return int(c - '0'), nil
		}
		if c >= 'A' && c <= 'Z' {
			return int(c-'A') + 10, nil
		}
		if c >= 'a' && c <= 'z' {
			return int(c-'a') + 10, nil
		}
	}
	
	// Try to parse as integer
	var id int
	_, err := fmt.Sscanf(s, "%d", &id)
	if err == nil && id >= 0 && id <= 35 {
		return id, nil
	}
	
	return -1, fmt.Errorf("invalid window number: %s", s)
}

