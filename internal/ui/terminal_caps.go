package ui

import (
	"os"
	"strings"
)

// TerminalCapabilities represents detected terminal features.
type TerminalCapabilities struct {
	HasColor         bool
	Supports256Color bool
	SupportsTrueColor bool
	SupportsMouse    bool
	SupportsBracketedPaste bool
	SupportsCursor   bool
	SupportsAltScreen bool
}

// DetectTerminalCapabilities determines capabilities using TERM/COLORTERM.
func DetectTerminalCapabilities() TerminalCapabilities {
	term := strings.ToLower(os.Getenv("TERM"))
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))

	caps := TerminalCapabilities{}
	if term != "" {
		caps.HasColor = strings.Contains(term, "color")
		caps.Supports256Color = strings.Contains(term, "256color")
		caps.SupportsCursor = true
		// Mouse and bracketed paste are typically supported by xterm-like terms.
		if strings.Contains(term, "xterm") || strings.Contains(term, "screen") || strings.Contains(term, "tmux") {
			caps.SupportsMouse = true
			caps.SupportsBracketedPaste = true
			caps.SupportsAltScreen = true
		}
	}

	if colorTerm != "" {
		caps.HasColor = true
		if strings.Contains(colorTerm, "truecolor") || strings.Contains(colorTerm, "24bit") {
			caps.SupportsTrueColor = true
		}
	}

	if caps.Supports256Color && caps.SupportsTrueColor {
		caps.HasColor = true
	}

	return caps
}

