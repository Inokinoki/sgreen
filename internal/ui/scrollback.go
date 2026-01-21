package ui

import (
	"bytes"
	"io"
	"sync"
)

// ScrollbackBuffer maintains a circular buffer of terminal output
type ScrollbackBuffer struct {
	mu       sync.RWMutex
	lines    [][]byte // Lines of text
	maxLines int      // Maximum number of lines
	size     int      // Current number of lines
	start    int      // Start index for circular buffer
}

// NewScrollbackBuffer creates a new scrollback buffer with the specified size
func NewScrollbackBuffer(maxLines int) *ScrollbackBuffer {
	if maxLines <= 0 {
		maxLines = 1000 // Default size
	}
	return &ScrollbackBuffer{
		lines:    make([][]byte, maxLines),
		maxLines: maxLines,
		size:     0,
		start:    0,
	}
}

// Append adds a line to the scrollback buffer
func (sb *ScrollbackBuffer) Append(line []byte) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Make a copy of the line
	lineCopy := make([]byte, len(line))
	copy(lineCopy, line)

	if sb.size < sb.maxLines {
		// Buffer not full yet
		sb.lines[sb.size] = lineCopy
		sb.size++
	} else {
		// Buffer is full, overwrite oldest line
		sb.lines[sb.start] = lineCopy
		sb.start = (sb.start + 1) % sb.maxLines
	}
}

// AppendBytes appends bytes to the current line or creates a new line
func (sb *ScrollbackBuffer) AppendBytes(data []byte) {
	// Split by newlines
	parts := bytes.Split(data, []byte{'\n'})
	for i, part := range parts {
		if i == 0 {
			// First part - append to last line if it exists
			if sb.size > 0 {
				lastIdx := (sb.start + sb.size - 1) % sb.maxLines
				sb.lines[lastIdx] = append(sb.lines[lastIdx], part...)
			} else {
				// No lines yet, create first line
				sb.Append(part)
			}
		} else if i == len(parts)-1 && len(part) == 0 {
			// Last empty part (trailing newline) - don't create empty line
			continue
		} else {
			// New line
			sb.Append(part)
		}
	}
}

// GetLine returns a specific line from the buffer
func (sb *ScrollbackBuffer) GetLine(index int) []byte {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if index < 0 || index >= sb.size {
		return nil
	}

	actualIdx := (sb.start + index) % sb.maxLines
	line := sb.lines[actualIdx]
	result := make([]byte, len(line))
	copy(result, line)
	return result
}

// GetLines returns a range of lines
func (sb *ScrollbackBuffer) GetLines(start, end int) [][]byte {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if start < 0 {
		start = 0
	}
	if end > sb.size {
		end = sb.size
	}
	if start >= end {
		return nil
	}

	result := make([][]byte, end-start)
	for i := start; i < end; i++ {
		actualIdx := (sb.start + i) % sb.maxLines
		line := sb.lines[actualIdx]
		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)
		result[i-start] = lineCopy
	}
	return result
}

// Size returns the current number of lines in the buffer
func (sb *ScrollbackBuffer) Size() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.size
}

// Clear clears the scrollback buffer
func (sb *ScrollbackBuffer) Clear() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.size = 0
	sb.start = 0
}

// WriteTo writes the entire scrollback buffer to a writer
func (sb *ScrollbackBuffer) WriteTo(w io.Writer) (int64, error) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	var total int64
	for i := 0; i < sb.size; i++ {
		actualIdx := (sb.start + i) % sb.maxLines
		n, err := w.Write(sb.lines[actualIdx])
		total += int64(n)
		if err != nil {
			return total, err
		}
		if i < sb.size-1 {
			// Add newline between lines (except after last line)
			n, err := w.Write([]byte{'\n'})
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
	}
	return total, nil
}
