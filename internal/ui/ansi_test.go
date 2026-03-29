package ui

import (
	"bytes"
	"testing"
)

func TestClearScreen(t *testing.T) {
	var buf bytes.Buffer
	ClearScreen(&buf)

	expected := "\033[2J"
	if buf.String() != expected {
		t.Errorf("ClearScreen() = %q, want %q", buf.String(), expected)
	}
}

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		name  string
		row   int
		col   int
		want  string
	}{
		{"valid position", 5, 10, "\033[5;10H"},
		{"row less than 1", 0, 10, "\033[1;10H"},
		{"col less than 1", 5, 0, "\033[5;1H"},
		{"both less than 1", 0, 0, "\033[1;1H"},
		{"large values", 100, 200, "\033[100;200H"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			MoveCursor(&buf, tt.row, tt.col)
			if buf.String() != tt.want {
				t.Errorf("MoveCursor(%d, %d) = %q, want %q", tt.row, tt.col, buf.String(), tt.want)
			}
		})
	}
}

func TestClearLine(t *testing.T) {
	var buf bytes.Buffer
	ClearLine(&buf)

	expected := "\033[K"
	if buf.String() != expected {
		t.Errorf("ClearLine() = %q, want %q", buf.String(), expected)
	}
}

func TestClearScreenAndHome(t *testing.T) {
	var buf bytes.Buffer
	ClearScreenAndHome(&buf)

	expected := "\033[2J\033[1;1H"
	if buf.String() != expected {
		t.Errorf("ClearScreenAndHome() = %q, want %q", buf.String(), expected)
	}
}

func TestSetColor(t *testing.T) {
	tests := []struct {
		name  string
		color int
		want  string
	}{
		{"color 0", 0, "\033[30m"},
		{"color 7", 7, "\033[37m"},
		{"color 4", 4, "\033[34m"},
		{"color less than 0", -1, "\033[30m"},
		{"color greater than 7", 10, "\033[37m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			SetColor(&buf, tt.color)
			if buf.String() != tt.want {
				t.Errorf("SetColor(%d) = %q, want %q", tt.color, buf.String(), tt.want)
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
		want      string
	}{
		{"no styles", false, false, false, "\033[0m"},
		{"bold only", true, false, false, "\033[0m\033[1m"},
		{"underline only", false, true, false, "\033[0m\033[4m"},
		{"italic only", false, false, true, "\033[0m\033[3m"},
		{"bold and underline", true, true, false, "\033[0m\033[1m\033[4m"},
		{"all styles", true, true, true, "\033[0m\033[1m\033[4m\033[3m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			SetStyle(&buf, tt.bold, tt.underline, tt.italic)
			if buf.String() != tt.want {
				t.Errorf("SetStyle(%v, %v, %v) = %q, want %q", tt.bold, tt.underline, tt.italic, buf.String(), tt.want)
			}
		})
	}
}

func TestSetColor256(t *testing.T) {
	tests := []struct {
		name  string
		color int
		want  string
	}{
		{"color 0", 0, "\033[38;5;0m"},
		{"color 255", 255, "\033[38;5;255m"},
		{"color 128", 128, "\033[38;5;128m"},
		{"color less than 0", -1, "\033[38;5;0m"},
		{"color greater than 255", 300, "\033[38;5;255m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			SetColor256(&buf, tt.color)
			if buf.String() != tt.want {
				t.Errorf("SetColor256(%d) = %q, want %q", tt.color, buf.String(), tt.want)
			}
		})
	}
}

func TestSetTrueColor(t *testing.T) {
	tests := []struct {
		name string
		r, g, b int
		want string
	}{
		{"black", 0, 0, 0, "\033[38;2;0;0;0m"},
		{"white", 255, 255, 255, "\033[38;2;255;255;255m"},
		{"red", 255, 0, 0, "\033[38;2;255;0;0m"},
		{"green", 0, 255, 0, "\033[38;2;0;255;0m"},
		{"blue", 0, 0, 255, "\033[38;2;0;0;255m"},
		{"purple", 128, 0, 128, "\033[38;2;128;0;128m"},
		{"r less than 0", -1, 100, 100, "\033[38;2;0;100;100m"},
		{"r greater than 255", 300, 100, 100, "\033[38;2;255;100;100m"},
		{"g less than 0", 100, -1, 100, "\033[38;2;100;0;100m"},
		{"g greater than 255", 100, 300, 100, "\033[38;2;100;255;100m"},
		{"b less than 0", 100, 100, -1, "\033[38;2;100;100;0m"},
		{"b greater than 255", 100, 100, 300, "\033[38;2;100;100;255m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			SetTrueColor(&buf, tt.r, tt.g, tt.b)
			if buf.String() != tt.want {
				t.Errorf("SetTrueColor(%d, %d, %d) = %q, want %q", tt.r, tt.g, tt.b, buf.String(), tt.want)
			}
		})
	}
}

