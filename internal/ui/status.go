package ui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/inoki/sgreen/internal/session"
)

// StatusLine displays a status line at the bottom of the terminal
type StatusLine struct {
	enabled      bool
	format       string
	lastUpdate   time.Time
	lastRendered string
}

// NewStatusLine creates a new status line
func NewStatusLine(enabled bool, format string) *StatusLine {
	return &StatusLine{
		enabled:    enabled,
		format:     format,
		lastUpdate: time.Now(),
	}
}

// Update updates the status line with current session/window information
func (sl *StatusLine) Update(out *os.File, sess *session.Session) {
	if !sl.enabled {
		return
	}

	win := sess.GetCurrentWindow()
	if win == nil {
		return
	}

	// Get terminal width
	width := 80 // Default
	if termWidth, _, err := getTerminalSize(out); err == nil {
		width = termWidth
	}

	// Build status string
	status := sl.buildStatusString(sess, win, width)
	if status == sl.lastRendered {
		// No change, skip redraw
		return
	}

	// Move cursor to bottom line and clear it
	MoveCursor(out, getTerminalHeight(out), 1)
	ClearLine(out)
	_, _ = fmt.Fprint(out, status)

	sl.lastUpdate = time.Now()
	sl.lastRendered = status
}

// buildStatusString builds the status line string
func (sl *StatusLine) buildStatusString(sess *session.Session, win *session.Window, width int) string {
	// Default format: [session] window title
	format := sl.format
	if format == "" {
		format = "[%S] %n %t"
	}

	result := ""
	i := 0
	for i < len(format) {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'S': // Session name
				result += sess.ID
			case 'n': // Window number
				result += win.Number
			case 't': // Window title
				if win.Title != "" {
					result += win.Title
				} else {
					result += win.CmdPath
				}
			case 'h': // Hostname (or hardstatus - screen uses 'h' for hardstatus, but we'll use 'H' for hostname)
				// In screen, '%h' is the stored hardstatus of the window
				// For now, we'll use the window title as hardstatus
				if win.Title != "" {
					result += win.Title
				} else {
					result += win.CmdPath
				}
			case 'H': // Hostname (alternative to 'h')
				hostname, _ := os.Hostname()
				result += hostname
			case 'w': // Window count
				result += fmt.Sprintf("%d", len(sess.Windows))
			case 'c': // Current window index
				result += fmt.Sprintf("%d", sess.CurrentWindow+1)
			case 'D': // Date (YYYY-MM-DD)
				result += time.Now().Format("2006-01-02")
			case 'T': // Time (HH:MM:SS)
				result += time.Now().Format("15:04:05")
			case 'l': // Load average
				loadStr := getLoadAverage()
				result += loadStr
			case '%': // Literal %
				result += "%"
			default:
				result += string(format[i+1])
			}
			i += 2
		} else {
			result += string(format[i])
			i++
		}
	}

	// Truncate to fit width
	if len(result) > width {
		result = result[:width-3] + "..."
	}

	return result
}

// getTerminalSize gets the terminal size
func getTerminalSize(file *os.File) (width, height int, err error) {
	return term.GetSize(int(file.Fd()))
}

// getTerminalHeight gets the terminal height
func getTerminalHeight(file *os.File) int {
	_, height, err := getTerminalSize(file)
	if err != nil {
		return 24
	}
	return height
}

// getLoadAverage gets the system load average
func getLoadAverage() string {
	if runtime.GOOS == "windows" {
		return "N/A"
	}

	// Try to read from /proc/loadavg on Linux
	if loadavg, err := os.ReadFile("/proc/loadavg"); err == nil {
		loadStr := strings.TrimSpace(string(loadavg))
		// Extract first value (1-minute load average)
		parts := strings.Fields(loadStr)
		if len(parts) > 0 {
			return parts[0]
		}
		return loadStr
	}

	// On other Unix systems, we could use syscall.Getloadavg if available
	// For now, return a placeholder
	return "N/A"
}

// ShowWindowList displays a list of windows
func ShowWindowList(out *os.File, sess *session.Session) {
	_, _ = fmt.Fprintf(out, "\r\nWindow List:\r\n")
	for i, win := range sess.Windows {
		marker := " "
		if i == sess.CurrentWindow {
			marker = "*"
		}
		title := win.Title
		if title == "" {
			title = win.CmdPath
		}
		_, _ = fmt.Fprintf(out, "%s %s: %s\r\n", marker, win.Number, title)
	}
	_, _ = fmt.Fprintf(out, "\r\nPress any key to continue...\r\n")
}

// ShowInteractiveWindowList displays an interactive window list for selection
func ShowInteractiveWindowList(in, out *os.File, sess *session.Session) error {
	// Display window list
	_, _ = fmt.Fprintf(out, "\r\nWindow List (select with number/name or arrow keys):\r\n")
	for i, win := range sess.Windows {
		marker := " "
		if i == sess.CurrentWindow {
			marker = "*"
		}
		title := win.Title
		if title == "" {
			title = win.CmdPath
		}
		_, _ = fmt.Fprintf(out, "%s %s: %s\r\n", marker, win.Number, title)
	}
	_, _ = fmt.Fprintf(out, "\r\nSelect window (number/name/Enter to cancel): ")

	// Read input
	buf := make([]byte, 1)
	var input []byte
	for {
		n, err := in.Read(buf)
		if err != nil || n == 0 {
			return nil
		}

		b := buf[0]

		// Handle Enter/Return
		if b == '\n' || b == '\r' {
			if len(input) == 0 {
				// Cancel - no input
				_, _ = fmt.Fprintf(out, "\r\n")
				return nil
			}
			break
		}

		// Handle Escape
		if b == 0x1b { // ESC
			_, _ = fmt.Fprintf(out, "\r\n")
			return nil
		}

		// Handle backspace
		if b == '\b' || b == 0x7f {
			if len(input) > 0 {
				input = input[:len(input)-1]
				_, _ = fmt.Fprintf(out, "\b \b")
			}
			continue
		}

		// Handle printable characters
		if b >= 32 && b < 127 {
			input = append(input, b)
			_, _ = fmt.Fprint(out, string(b))
		}
	}

	// Parse input
	selection := strings.TrimSpace(string(input))
	if selection == "" {
		return nil
	}

	// Try to switch to selected window
	err := sess.SwitchToWindow(selection)
	if err != nil {
		_, _ = fmt.Fprintf(out, "\r\nInvalid window: %s\r\n", selection)
		// Wait a bit for user to see error
		time.Sleep(1 * time.Second)
		return nil
	}

	_, _ = fmt.Fprintf(out, "\r\n")
	return nil
}
