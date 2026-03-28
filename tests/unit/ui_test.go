package unit

import (
	"bytes"
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
