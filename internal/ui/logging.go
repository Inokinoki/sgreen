package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogWriter wraps a file with timestamping and rotation support
type LogWriter struct {
	file        *os.File
	mu          sync.Mutex
	basePath    string
	maxSize     int64
	currentSize int64
	timestamp   bool
}

// NewLogWriter creates a new log writer with optional timestamping
func NewLogWriter(filepath string, timestamp bool) (*LogWriter, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// Get current file size
	stat, err := file.Stat()
	var currentSize int64
	if err == nil {
		currentSize = stat.Size()
	}

	return &LogWriter{
		file:        file,
		basePath:    filepath,
		maxSize:     10 * 1024 * 1024, // 10MB default
		currentSize: currentSize,
		timestamp:   timestamp,
	}, nil
}

// Write writes data to the log file with optional timestamping
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// Check if rotation is needed
	if lw.maxSize > 0 && lw.currentSize+int64(len(p)) > lw.maxSize {
		if err := lw.rotate(); err != nil {
			// Non-fatal, continue with current file
		}
	}

	// Add timestamp if enabled
	if lw.timestamp {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000 ")
		if _, err := lw.file.WriteString(timestamp); err != nil {
			return 0, err
		}
	}

	// Write the data
	n, err = lw.file.Write(p)
	if err == nil {
		lw.currentSize += int64(n)
	}
	return n, err
}

// rotate rotates the log file
func (lw *LogWriter) rotate() error {
	// Close current file
	lw.file.Close()

	// Rename current file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := lw.basePath + "." + timestamp
	os.Rename(lw.basePath, rotatedPath)

	// Open new file
	file, err := os.OpenFile(lw.basePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	lw.file = file
	lw.currentSize = 0
	return nil
}

// Close closes the log file
func (lw *LogWriter) Close() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.file.Close()
}

// SetMaxSize sets the maximum log file size before rotation
func (lw *LogWriter) SetMaxSize(size int64) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	lw.maxSize = size
}

// PerWindowLogWriter manages per-window logging
type PerWindowLogWriter struct {
	writers   map[int]*LogWriter
	mu        sync.RWMutex
	baseDir   string
	timestamp bool
}

// NewPerWindowLogWriter creates a new per-window log writer
func NewPerWindowLogWriter(baseDir string, timestamp bool) *PerWindowLogWriter {
	return &PerWindowLogWriter{
		writers:   make(map[int]*LogWriter),
		baseDir:   baseDir,
		timestamp: timestamp,
	}
}

// GetWriter gets or creates a log writer for a window
func (pwlw *PerWindowLogWriter) GetWriter(windowID int, windowTitle string) (io.Writer, error) {
	pwlw.mu.Lock()
	defer pwlw.mu.Unlock()

	if writer, exists := pwlw.writers[windowID]; exists {
		return writer, nil
	}

	// Create log file name based on window ID and title
	var filename string
	if windowTitle != "" {
		// Sanitize title for filename
		safeTitle := sanitizeFilename(windowTitle)
		filename = fmt.Sprintf("window-%d-%s.log", windowID, safeTitle)
	} else {
		filename = fmt.Sprintf("window-%d.log", windowID)
	}

	logPath := filepath.Join(pwlw.baseDir, filename)
	writer, err := NewLogWriter(logPath, pwlw.timestamp)
	if err != nil {
		return nil, err
	}

	pwlw.writers[windowID] = writer
	return writer, nil
}

// Close closes all log writers
func (pwlw *PerWindowLogWriter) Close() error {
	pwlw.mu.Lock()
	defer pwlw.mu.Unlock()

	var lastErr error
	for _, writer := range pwlw.writers {
		if err := writer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// sanitizeFilename sanitizes a string for use in a filename
func sanitizeFilename(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result = append(result, r)
		} else if r == ' ' {
			result = append(result, '-')
		}
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return string(result)
}
