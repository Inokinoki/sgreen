package ui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// ShowStartupMessage displays startup message
func ShowStartupMessage(out io.Writer, sessName string, windowCount int) {
	message := "\r\n*** Welcome to sgreen ***\r\n"
	message += fmt.Sprintf("Session: %s\r\n", sessName)
	message += fmt.Sprintf("Windows: %d\r\n", windowCount)
	message += "Press Ctrl+A ? for help\r\n"
	message += "\r\n"
	_, _ = fmt.Fprint(out, message)
}

// ShowBell displays a bell (audible or visual)
func ShowBell(out io.Writer, visual bool) {
	if visual {
		// Visual bell: flash screen
		_, _ = fmt.Fprintf(out, "\033[?5h") // Turn on reverse video
		time.Sleep(50 * time.Millisecond)
		_, _ = fmt.Fprintf(out, "\033[?5l") // Turn off reverse video
	} else {
		// Audible bell
		_, _ = fmt.Fprintf(out, "\a")
	}
}

// ShowMessage displays a message to user
func ShowMessage(out io.Writer, message string) {
	// Clear current line and show message
	_, _ = fmt.Fprintf(out, "\r\033[K%s\r\n", message)
}

// ShowActivityMessage shows an activity notification
func ShowActivityMessage(out io.Writer, windowTitle string) {
	message := fmt.Sprintf("Activity in window: %s", windowTitle)
	ShowMessage(out, message)
}

// ShowSilenceMessage shows a silence notification
func ShowSilenceMessage(out io.Writer, windowTitle string) {
	message := fmt.Sprintf("Silence in window: %s", windowTitle)
	ShowMessage(out, message)
}

// ShowVersion displays version information
func ShowVersion(out io.Writer) {
	message := "\r\n*** sgreen version 0.1.0 ***\r\n"
	message += "A simplified screen-like terminal multiplexer\r\n"
	message += "Compatible with GNU screen command-line interface\r\n"
	message += "\r\nPress any key to continue...\r\n"
	_, _ = fmt.Fprint(out, message)
}

// ShowLicense displays license information
func ShowLicense(out io.Writer) {
	message := "\r\n*** sgreen License ***\r\n"
	message += "sgreen is open source software.\r\n"
	message += "See LICENSE file for details.\r\n"
	message += "\r\nPress any key to continue...\r\n"
	_, _ = fmt.Fprint(out, message)
}

// ShowTimeLoad displays time and load average
func ShowTimeLoad(out io.Writer) {
	now := time.Now()
	message := fmt.Sprintf("\r\nTime: %s\r\n", now.Format("2006-01-02 15:04:05"))

	// Try to get load average (Unix only)
	if runtime.GOOS != "windows" {
		// Read from /proc/loadavg on Linux, or use syscall on other Unix
		if loadavg, err := os.ReadFile("/proc/loadavg"); err == nil {
			loadStr := strings.TrimSpace(string(loadavg))
			message += fmt.Sprintf("Load: %s\r\n", loadStr)
		}
	}

	message += "\r\nPress any key to continue...\r\n"
	_, _ = fmt.Fprint(out, message)
}

// BlankScreen clears terminal display
func BlankScreen(out io.Writer) {
	// Clear screen and move cursor to top
	_, _ = fmt.Fprintf(out, "\033[2J\033[H")
}
