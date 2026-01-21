package ui

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// ShowStartupMessage displays the startup message
func ShowStartupMessage(out *os.File, sessName string, windowCount int) {
	message := fmt.Sprintf("\r\n*** Welcome to sgreen ***\r\n")
	message += fmt.Sprintf("Session: %s\r\n", sessName)
	message += fmt.Sprintf("Windows: %d\r\n", windowCount)
	message += fmt.Sprintf("Press Ctrl+A ? for help\r\n")
	message += fmt.Sprintf("\r\n")
	fmt.Fprint(out, message)
}

// ShowBell displays a bell (audible or visual)
func ShowBell(out *os.File, visual bool) {
	if visual {
		// Visual bell: flash screen
		fmt.Fprint(out, "\033[?5h") // Turn on reverse video
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(out, "\033[?5l") // Turn off reverse video
	} else {
		// Audible bell
		fmt.Fprint(out, "\a")
	}
}

// ShowMessage displays a message to the user
func ShowMessage(out *os.File, message string) {
	// Clear current line and show message
	fmt.Fprintf(out, "\r\033[K%s\r\n", message)
}

// ShowActivityMessage shows an activity notification
func ShowActivityMessage(out *os.File, windowTitle string) {
	message := fmt.Sprintf("Activity in window: %s", windowTitle)
	ShowMessage(out, message)
}

// ShowSilenceMessage shows a silence notification
func ShowSilenceMessage(out *os.File, windowTitle string) {
	message := fmt.Sprintf("Silence in window: %s", windowTitle)
	ShowMessage(out, message)
}

// ShowVersion displays version information
func ShowVersion(out *os.File) {
	message := "\r\n*** sgreen version 0.1.0 ***\r\n"
	message += "A simplified screen-like terminal multiplexer\r\n"
	message += "Compatible with GNU screen command-line interface\r\n"
	message += "\r\nPress any key to continue...\r\n"
	fmt.Fprint(out, message)
}

// ShowLicense displays license information
func ShowLicense(out *os.File) {
	message := "\r\n*** sgreen License ***\r\n"
	message += "sgreen is open source software.\r\n"
	message += "See LICENSE file for details.\r\n"
	message += "\r\nPress any key to continue...\r\n"
	fmt.Fprint(out, message)
}

// ShowTimeLoad displays time and load average
func ShowTimeLoad(out *os.File) {
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
	fmt.Fprint(out, message)
}

// BlankScreen clears the terminal display
func BlankScreen(out *os.File) {
	// Clear screen and move cursor to top
	fmt.Fprint(out, "\033[2J\033[H")
}

