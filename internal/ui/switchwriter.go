package ui

import (
	"io"
	"sync"
)

// switchWriter is a write-through whose destination can be swapped as the
// attach loop moves between windows; it lets one persistent input
// goroutine feed whichever window is current.
type switchWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// Set replaces the write destination.
func (s *switchWriter) Set(w io.Writer) {
	s.mu.Lock()
	s.w = w
	s.mu.Unlock()
}

// Write implements io.Writer against the current destination.
func (s *switchWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	w := s.w
	s.mu.Unlock()
	if w == nil {
		return len(p), nil // drop until a window is attached
	}
	return w.Write(p)
}
