package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/inoki/sgreen/internal/ui"
)

// The tests in this file are the non-duplicate survivors of
// tests/unit/ui_test.go; the duplicated ones (encoding, attach-config,
// terminal capabilities) already exist in the in-package test files.

func TestEncodingWriter(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		data     string
	}{
		{name: "UTF-8 encoding", encoding: "UTF-8", data: "Hello, World!"},
		{name: "Empty encoding", encoding: "", data: "Test data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := ui.WrapEncodingWriter(&buf, tt.encoding)
			if writer == nil {
				t.Fatal("WrapEncodingWriter returned nil")
			}
			if _, err := writer.Write([]byte(tt.data)); err != nil {
				t.Fatalf("write via encoding writer: %v", err)
			}
			if buf.String() != tt.data {
				t.Errorf("data altered: got %q, want %q", buf.String(), tt.data)
			}
		})
	}
}

func TestPasteBuffer(t *testing.T) {
	content := []byte("test content")
	ui.SetPasteBuffer(content)

	if retrieved := ui.GetPasteBuffer(); string(retrieved) != string(content) {
		t.Errorf("GetPasteBuffer() = %q, want %q", retrieved, content)
	}
}

func TestPasteBufferEmpty(t *testing.T) {
	ui.SetPasteBuffer([]byte{})

	if retrieved := ui.GetPasteBuffer(); len(retrieved) != 0 {
		t.Errorf("GetPasteBuffer() should be empty after setting empty buffer, got %q", retrieved)
	}
}

func TestShowHelp(t *testing.T) {
	var buf bytes.Buffer
	ui.ShowHelp(&buf)

	result := buf.String()
	if result == "" {
		t.Fatal("ShowHelp should produce output")
	}
	if !strings.Contains(result, "sgreen Key Bindings") {
		t.Errorf("ShowHelp output should contain 'sgreen Key Bindings'")
	}
}
