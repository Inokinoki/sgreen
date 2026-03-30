package unit

import (
	"bytes"
	"testing"

	"github.com/inoki/sgreen/internal/ui"
)

func TestClearScreen(t *testing.T) {
	var buf bytes.Buffer
	ui.ClearScreen(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("ClearScreen should produce output")
	}

	if result != "\033[2J" {
		t.Errorf("ClearScreen output mismatch")
	}
}

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		name     string
		row      int
		col      int
		expected string
	}{
		{"home position", 1, 1, "\033[1;1H"},
		{"middle position", 10, 20, "\033[10;20H"},
		{"large position", 100, 200, "\033[100;200H"},
		{"zero row clamped", 0, 10, "\033[1;10H"},
		{"zero col clamped", 10, 0, "\033[10;1H"},
		{"both zero clamped", 0, 0, "\033[1;1H"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.MoveCursor(&buf, tt.row, tt.col)

			result := buf.String()
			if result != tt.expected {
				t.Errorf("MoveCursor(%d, %d) = %q, want %q", tt.row, tt.col, result, tt.expected)
			}
		})
	}
}

func TestClearLine(t *testing.T) {
	var buf bytes.Buffer
	ui.ClearLine(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("ClearLine should produce output")
	}

	if result != "\033[K" {
		t.Errorf("ClearLine output mismatch")
	}
}

func TestClearScreenAndHome(t *testing.T) {
	var buf bytes.Buffer
	ui.ClearScreenAndHome(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("ClearScreenAndHome should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("\033[2J")) {
		t.Errorf("Should contain clear screen sequence")
	}

	if !bytes.Contains(buf.Bytes(), []byte("\033[1;1H")) {
		t.Errorf("Should contain move cursor home sequence")
	}
}

func TestSetColor(t *testing.T) {
	tests := []struct {
		name     string
		color    int
		expected string
	}{
		{"black", 0, "\033[30m"},
		{"red", 1, "\033[31m"},
		{"green", 2, "\033[32m"},
		{"yellow", 3, "\033[33m"},
		{"blue", 4, "\033[34m"},
		{"magenta", 5, "\033[35m"},
		{"cyan", 6, "\033[36m"},
		{"white", 7, "\033[37m"},
		{"negative clamped", -1, "\033[30m"},
		{"too high clamped", 10, "\033[37m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.SetColor(&buf, tt.color)

			result := buf.String()
			if result != tt.expected {
				t.Errorf("SetColor(%d) = %q, want %q", tt.color, result, tt.expected)
			}
		})
	}
}

func TestSetStyle(t *testing.T) {
	tests := []struct {
		name      string
		bold      bool
		underline bool
		italic    bool
	}{
		{"none", false, false, false},
		{"bold", true, false, false},
		{"underline", false, true, false},
		{"italic", false, false, true},
		{"bold underline", true, true, false},
		{"bold italic", true, false, true},
		{"underline italic", false, true, true},
		{"all styles", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.SetStyle(&buf, tt.bold, tt.underline, tt.italic)

			result := buf.String()
			if result == "" {
				t.Errorf("SetStyle should produce output")
			}

			if !bytes.Contains(buf.Bytes(), []byte("\033[0m")) {
				t.Errorf("SetStyle should always start with reset")
			}

			if tt.bold && !bytes.Contains(buf.Bytes(), []byte("\033[1m")) {
				t.Errorf("SetStyle with bold should contain bold sequence")
			}

			if tt.underline && !bytes.Contains(buf.Bytes(), []byte("\033[4m")) {
				t.Errorf("SetStyle with underline should contain underline sequence")
			}

			if tt.italic && !bytes.Contains(buf.Bytes(), []byte("\033[3m")) {
				t.Errorf("SetStyle with italic should contain italic sequence")
			}
		})
	}
}

func TestSetColor256(t *testing.T) {
	tests := []struct {
		name     string
		color    int
		expected string
	}{
		{"color 0", 0, "\033[38;5;0m"},
		{"color 128", 128, "\033[38;5;128m"},
		{"color 255", 255, "\033[38;5;255m"},
		{"negative clamped", -1, "\033[38;5;0m"},
		{"too high clamped", 300, "\033[38;5;255m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.SetColor256(&buf, tt.color)

			result := buf.String()
			if result != tt.expected {
				t.Errorf("SetColor256(%d) = %q, want %q", tt.color, result, tt.expected)
			}
		})
	}
}

func TestSetTrueColor(t *testing.T) {
	tests := []struct {
		name     string
		r        int
		g        int
		b        int
		expected string
	}{
		{"black", 0, 0, 0, "\033[38;2;0;0;0m"},
		{"red", 255, 0, 0, "\033[38;2;255;0;0m"},
		{"green", 0, 255, 0, "\033[38;2;0;255;0m"},
		{"blue", 0, 0, 255, "\033[38;2;0;0;255m"},
		{"white", 255, 255, 255, "\033[38;2;255;255;255m"},
		{"gray", 128, 128, 128, "\033[38;2;128;128;128m"},
		{"negative clamped", -1, -1, -1, "\033[38;2;0;0;0m"},
		{"too high clamped", 300, 300, 300, "\033[38;2;255;255;255m"},
		{"mixed", 100, 200, 50, "\033[38;2;100;200;50m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.SetTrueColor(&buf, tt.r, tt.g, tt.b)

			result := buf.String()
			if result != tt.expected {
				t.Errorf("SetTrueColor(%d, %d, %d) = %q, want %q", tt.r, tt.g, tt.b, result, tt.expected)
			}
		})
	}
}
