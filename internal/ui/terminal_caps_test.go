package ui

import (
	"os"
	"testing"
)

func TestDetectTerminalCapabilities(t *testing.T) {
	tests := []struct {
		name      string
		term      string
		colorterm string
		wantColor bool
	}{
		{"xterm-256color", "xterm-256color", "", true},
		{"screen-256color", "screen-256color", "", true},
		{"xterm", "xterm", "", false},
		{"dumb", "dumb", "", false},
		{"truecolor term", "xterm", "truecolor", true},
		{"24bit term", "xterm", "24bit", true},
		{"empty term", "", "truecolor", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TERM", tt.term)
			os.Setenv("COLORTERM", tt.colorterm)
			caps := DetectTerminalCapabilities()

			if caps.HasColor != tt.wantColor {
				t.Errorf("DetectTerminalCapabilities() HasColor = %v, want %v", caps.HasColor, tt.wantColor)
			}
		})
	}
}
