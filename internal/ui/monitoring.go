package ui

import (
	"sync"
	"time"

	"github.com/inoki/sgreen/internal/session"
)

// ActivityMonitor monitors activity in windows
type ActivityMonitor struct {
	mu              sync.RWMutex
	enabled         bool
	message         string
	lastActivity    map[int]time.Time
	monitoredWindows map[int]bool
	activityChan    chan int
}

// SilenceMonitor monitors silence in windows
type SilenceMonitor struct {
	mu              sync.RWMutex
	enabled         bool
	message         string
	lastActivity    map[int]time.Time
	monitoredWindows map[int]bool
	silenceTimeout  time.Duration
	silenceChan     chan int
}

// NewActivityMonitor creates a new activity monitor
func NewActivityMonitor(message string) *ActivityMonitor {
	return &ActivityMonitor{
		enabled:         false,
		message:         message,
		lastActivity:    make(map[int]time.Time),
		monitoredWindows: make(map[int]bool),
		activityChan:    make(chan int, 10),
	}
}

// NewSilenceMonitor creates a new silence monitor
func NewSilenceMonitor(message string, timeout time.Duration) *SilenceMonitor {
	return &SilenceMonitor{
		enabled:         false,
		message:         message,
		lastActivity:    make(map[int]time.Time),
		monitoredWindows: make(map[int]bool),
		silenceTimeout:  timeout,
		silenceChan:     make(chan int, 10),
	}
}

// Enable enables activity monitoring
func (am *ActivityMonitor) Enable() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.enabled = true
}

// Disable disables activity monitoring
func (am *ActivityMonitor) Disable() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.enabled = false
}

// MonitorWindow enables monitoring for a specific window
func (am *ActivityMonitor) MonitorWindow(windowID int) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.monitoredWindows[windowID] = true
	am.lastActivity[windowID] = time.Now()
}

// UnmonitorWindow disables monitoring for a specific window
func (am *ActivityMonitor) UnmonitorWindow(windowID int) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.monitoredWindows, windowID)
	delete(am.lastActivity, windowID)
}

// RecordActivity records activity in a window
func (am *ActivityMonitor) RecordActivity(windowID int) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if !am.enabled {
		return
	}
	
	if !am.monitoredWindows[windowID] {
		return
	}
	
	// Check if this is activity in a background window
	// (not the current window)
	am.lastActivity[windowID] = time.Now()
	
	// Send notification if window is monitored
	select {
	case am.activityChan <- windowID:
	default:
		// Channel full, drop notification
	}
}

// GetActivityChannel returns the channel for activity notifications
func (am *ActivityMonitor) GetActivityChannel() <-chan int {
	return am.activityChan
}

// GetMessage returns the activity message template
func (am *ActivityMonitor) GetMessage() string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	if am.message == "" {
		return "Activity in window %n"
	}
	return am.message
}

// Enable enables silence monitoring
func (sm *SilenceMonitor) Enable() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.enabled = true
}

// Disable disables silence monitoring
func (sm *SilenceMonitor) Disable() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.enabled = false
}

// MonitorWindow enables monitoring for a specific window
func (sm *SilenceMonitor) MonitorWindow(windowID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.monitoredWindows[windowID] = true
	sm.lastActivity[windowID] = time.Now()
}

// UnmonitorWindow disables monitoring for a specific window
func (sm *SilenceMonitor) UnmonitorWindow(windowID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.monitoredWindows, windowID)
	delete(sm.lastActivity, windowID)
}

// RecordActivity records activity in a window
func (sm *SilenceMonitor) RecordActivity(windowID int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.lastActivity[windowID] = time.Now()
}

// StartMonitoring starts the silence monitoring loop
func (sm *SilenceMonitor) StartMonitoring(currentWindowID func() int) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			sm.mu.RLock()
			if !sm.enabled {
				sm.mu.RUnlock()
				continue
			}
			
			currentWin := currentWindowID()
			now := time.Now()
			
			for winID := range sm.monitoredWindows {
				if winID == currentWin {
					// Don't monitor current window
					continue
				}
				
				lastAct, exists := sm.lastActivity[winID]
				if !exists {
					sm.lastActivity[winID] = now
					continue
				}
				
				if now.Sub(lastAct) > sm.silenceTimeout {
					// Window has been silent
					select {
					case sm.silenceChan <- winID:
					default:
						// Channel full, drop notification
					}
					// Reset to avoid repeated notifications
					sm.lastActivity[winID] = now
				}
			}
			sm.mu.RUnlock()
		}
	}()
}

// GetSilenceChannel returns the channel for silence notifications
func (sm *SilenceMonitor) GetSilenceChannel() <-chan int {
	return sm.silenceChan
}

// GetMessage returns the silence message template
func (sm *SilenceMonitor) GetMessage() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.message == "" {
		return "Silence in window %n"
	}
	return sm.message
}

// FormatMessage formats a message template with window information
func FormatMessage(template string, win *session.Window) string {
	result := ""
	i := 0
	for i < len(template) {
		if template[i] == '%' && i+1 < len(template) {
			switch template[i+1] {
			case 'n':
				// Window number
				result += win.Number
			case 't':
				// Window title
				if win.Title != "" {
					result += win.Title
				} else {
					result += win.CmdPath
				}
			case 'G':
				// Bell character
				result += "\a"
			case '%':
				// Literal %
				result += "%"
			default:
				result += string(template[i+1])
			}
			i += 2
		} else {
			result += string(template[i])
			i++
		}
	}
	return result
}

