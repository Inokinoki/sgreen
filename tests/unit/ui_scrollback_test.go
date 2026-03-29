package unit

import (
	"bytes"
	"testing"

	"github.com/inoki/sgreen/internal/ui"
)

func TestNewScrollbackBuffer(t *testing.T) {
	tests := []struct {
		name     string
		maxLines int
	}{
		{"default size", 1000},
		{"custom size", 100},
		{"large size", 10000},
		{"small size", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := ui.NewScrollbackBuffer(tt.maxLines)
			if sb == nil {
				t.Errorf("NewScrollbackBuffer should not return nil")
			}
			if sb.Size() != 0 {
				t.Errorf("New buffer should be empty, got size %d", sb.Size())
			}
		})
	}
}

func TestNewScrollbackBufferZero(t *testing.T) {
	sb := ui.NewScrollbackBuffer(0)
	if sb == nil {
		t.Errorf("NewScrollbackBuffer with 0 should use default size")
	}
}

func TestScrollbackBufferAppend(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)
	testLine := []byte("test line")

	sb.Append(testLine)

	if sb.Size() != 1 {
		t.Errorf("Expected size 1, got %d", sb.Size())
	}

	retrieved := sb.GetLine(0)
	if !bytes.Equal(retrieved, testLine) {
		t.Errorf("Retrieved line mismatch")
	}
}

func TestScrollbackBufferAppendMultiple(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)

	for i := 0; i < 5; i++ {
		line := []byte(string(rune('A' + i)))
		sb.Append(line)
	}

	if sb.Size() != 5 {
		t.Errorf("Expected size 5, got %d", sb.Size())
	}

	for i := 0; i < 5; i++ {
		line := sb.GetLine(i)
		expected := []byte(string(rune('A' + i)))
		if !bytes.Equal(line, expected) {
			t.Errorf("Line %d mismatch", i)
		}
	}
}

func TestScrollbackBufferOverflow(t *testing.T) {
	sb := ui.NewScrollbackBuffer(3)

	sb.Append([]byte("line1"))
	sb.Append([]byte("line2"))
	sb.Append([]byte("line3"))
	sb.Append([]byte("line4"))

	if sb.Size() != 3 {
		t.Errorf("Expected size 3 after overflow, got %d", sb.Size())
	}

	line1 := sb.GetLine(0)
	if bytes.Equal(line1, []byte("line1")) {
		t.Errorf("First line should be overwritten")
	}
}

func TestScrollbackBufferGetLineInvalid(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)
	sb.Append([]byte("test"))

	tests := []struct {
		name  string
		index int
	}{
		{"negative index", -1},
		{"index out of bounds", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := sb.GetLine(tt.index)
			if line != nil {
				t.Errorf("GetLine with invalid index should return nil")
			}
		})
	}
}

func TestScrollbackBufferGetLines(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)

	for i := 0; i < 5; i++ {
		sb.Append([]byte(string(rune('A' + i))))
	}

	lines := sb.GetLines(1, 3)
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	if !bytes.Equal(lines[0], []byte("B")) {
		t.Errorf("First line mismatch")
	}
	if !bytes.Equal(lines[1], []byte("C")) {
		t.Errorf("Second line mismatch")
	}
}

func TestScrollbackBufferGetLinesInvalid(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)
	sb.Append([]byte("test"))

	tests := []struct {
		name   string
		start  int
		end    int
		expectNil bool
	}{
		{"start >= end", 2, 2, true},
		{"negative start", -1, 2, false},
		{"end beyond size", 0, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := sb.GetLines(tt.start, tt.end)
			if tt.expectNil && lines != nil {
				t.Errorf("Expected nil for invalid range")
			}
		})
	}
}

func TestScrollbackBufferClear(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)

	for i := 0; i < 5; i++ {
		sb.Append([]byte(string(rune('A' + i))))
	}

	if sb.Size() != 5 {
		t.Errorf("Expected size 5 before clear")
	}

	sb.Clear()

	if sb.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", sb.Size())
	}

	line := sb.GetLine(0)
	if line != nil {
		t.Errorf("GetLine after clear should return nil")
	}
}

func TestScrollbackBufferAppendBytes(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)

	sb.AppendBytes([]byte("line1\nline2\nline3"))

	if sb.Size() != 3 {
		t.Errorf("Expected 3 lines, got %d", sb.Size())
	}

	if !bytes.Equal(sb.GetLine(0), []byte("line1")) {
		t.Errorf("First line mismatch")
	}
	if !bytes.Equal(sb.GetLine(1), []byte("line2")) {
		t.Errorf("Second line mismatch")
	}
	if !bytes.Equal(sb.GetLine(2), []byte("line3")) {
		t.Errorf("Third line mismatch")
	}
}

func TestScrollbackBufferAppendBytesPartial(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)

	sb.AppendBytes([]byte("line1"))
	sb.AppendBytes([]byte("partial"))
	sb.AppendBytes([]byte(" continuation\nline2"))

	if sb.Size() != 2 {
		t.Errorf("Expected 2 lines, got %d", sb.Size())
	}

	line1 := sb.GetLine(0)
	expectedLine1 := []byte("line1partial continuation")
	if !bytes.Equal(line1, expectedLine1) {
		t.Errorf("First line should be concatenated")
	}
}

func TestScrollbackBufferWriteTo(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)

	sb.Append([]byte("line1"))
	sb.Append([]byte("line2"))
	sb.Append([]byte("line3"))

	var buf bytes.Buffer
	n, err := sb.WriteTo(&buf)

	if err != nil {
		t.Errorf("WriteTo should not return error: %v", err)
	}

	if n <= 0 {
		t.Errorf("WriteTo should write bytes")
	}

	result := buf.String()
	if result == "" {
		t.Errorf("WriteTo should produce output")
	}
}

func TestScrollbackBufferWriteToEmpty(t *testing.T) {
	sb := ui.NewScrollbackBuffer(10)

	var buf bytes.Buffer
	n, err := sb.WriteTo(&buf)

	if err != nil {
		t.Errorf("WriteTo on empty buffer should not return error: %v", err)
	}

	if n != 0 {
		t.Errorf("WriteTo on empty buffer should write 0 bytes, got %d", n)
	}

	result := buf.String()
	if result != "" {
		t.Errorf("WriteTo on empty buffer should produce no output")
	}
}

