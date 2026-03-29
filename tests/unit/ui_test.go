package unit

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/inoki/sgreen/internal/ui"
)

func TestNormalizeEncoding(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"utf-8", "UTF-8"},
		{"UTF-8", "UTF-8"},
		{"utf8", "UTF8"},
		{" iso-8859-1 ", "ISO-8859-1"},
		{"ISO_8859_1", "ISO-8859-1"},
		{"latin1", "LATIN1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ui.NormalizeEncoding(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeEncoding(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsUTF8Encoding(t *testing.T) {
	tests := []struct {
		encoding string
		expected bool
	}{
		{"", true},
		{"UTF-8", true},
		{"utf8", true},
		{"UTF8", true},
		{"ISO-8859-1", false},
		{"latin1", false},
		{"windows-1252", false},
	}

	for _, tt := range tests {
		t.Run(tt.encoding, func(t *testing.T) {
			result := ui.IsUTF8Encoding(tt.encoding)
			if result != tt.expected {
				t.Errorf("IsUTF8Encoding(%q) = %v, want %v", tt.encoding, result, tt.expected)
			}
		})
	}
}

func TestConvertToUTF8(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		input    []byte
	}{
		{
			name:     "UTF-8 should pass through",
			encoding: "UTF-8",
			input:    []byte("Hello, 世界"),
		},
		{
			name:     "Empty encoding should pass through",
			encoding: "",
			input:    []byte("Hello"),
		},
		{
			name:     "ISO-8859-1 conversion",
			encoding: "ISO-8859-1",
			input:    []byte{0xE9, 0x20}, // é
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.ConvertToUTF8(tt.encoding, tt.input)
			if result == nil {
				t.Errorf("ConvertToUTF8 returned nil")
			}
		})
	}
}

func TestEncodingWriter(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		data     string
	}{
		{
			name:     "UTF-8 encoding",
			encoding: "UTF-8",
			data:     "Hello, World!",
		},
		{
			name:     "Empty encoding",
			encoding: "",
			data:     "Test data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := ui.WrapEncodingWriter(&buf, tt.encoding)
			if writer == nil {
				t.Fatalf("WrapEncodingWriter returned nil")
			}

			n, err := writer.Write([]byte(tt.data))
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			if n != len(tt.data) {
				t.Errorf("Write returned %d, want %d", n, len(tt.data))
			}

			result := buf.String()
			if result != tt.data {
				t.Errorf("Got %q, want %q", result, tt.data)
			}
		})
	}
}

func TestGetCharmap(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		expected bool
	}{
		{"ISO-8859-1", "ISO-8859-1", true},
		{"ISO8859-2", "ISO8859-2", true},
		{"latin9", "LATIN9", true},
		{"windows-1252", "WINDOWS-1252", true},
		{"cp1251", "CP1251", true},
		{"koi8-r", "KOI8-R", true},
		{"koi8-u", "KOI8-U", true},
		{"unknown", "UNKNOWN", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.GetCharmap(tt.encoding)
			hasResult := result != nil
			if hasResult != tt.expected {
				t.Errorf("GetCharmap(%q) returned nil=%v, want %v", tt.encoding, !hasResult, tt.expected)
			}
		})
	}
}

func TestDefaultAttachConfig(t *testing.T) {
	config := ui.DefaultAttachConfig()
	if config == nil {
		t.Errorf("DefaultAttachConfig() returned nil")
	}

	if config.CommandChar != 0x01 {
		t.Errorf("Expected CommandChar 0x01, got %v", config.CommandChar)
	}

	if config.LiteralChar != 'a' {
		t.Errorf("Expected LiteralChar 'a', got %v", config.LiteralChar)
	}

	if config.FlowControl != "off" {
		t.Errorf("Expected FlowControl off, got %s", config.FlowControl)
	}

	if config.Scrollback != 1000 {
		t.Errorf("Expected Scrollback 1000, got %d", config.Scrollback)
	}
}

func TestPasteBuffer(t *testing.T) {
	content := []byte("test content")
	ui.SetPasteBuffer(content)

	retrieved := ui.GetPasteBuffer()
	if string(retrieved) != string(content) {
		t.Errorf("GetPasteBuffer() = %q, want %q", string(retrieved), string(content))
	}
}

func TestPasteBufferEmpty(t *testing.T) {
	ui.SetPasteBuffer([]byte{})

	retrieved := ui.GetPasteBuffer()
	if len(retrieved) != 0 {
		t.Errorf("GetPasteBuffer() should return empty after setting empty buffer, got %q", string(retrieved))
	}
}

func TestShowHelp(t *testing.T) {
	var buf bytes.Buffer
	ui.ShowHelp(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowHelp should produce output")
	}

	if !strings.Contains(result, "sgreen Key Bindings") {
		t.Errorf("ShowHelp output should contain 'sgreen Key Bindings'")
	}
}

func TestDetectTerminalCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		term     string
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
			caps := ui.DetectTerminalCapabilities()

			if caps.HasColor != tt.wantColor {
				t.Errorf("DetectTerminalCapabilities() HasColor = %v, want %v", caps.HasColor, tt.wantColor)
			}
		})
	}
}
