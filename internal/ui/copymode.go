package ui

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"

	"github.com/inoki/sgreen/internal/session"
)

// CopyMode represents the copy mode state
type CopyMode struct {
	buffer        *ScrollbackBuffer
	startLine     int
	startCol      int
	endLine       int
	endCol        int
	currentLine   int
	currentCol    int
	selecting     bool
	selected      bool
	searchMode    bool
	searchTerm    string
	searchResults []int // Line numbers matching search
	searchIndex   int   // Current search result index
}

// PasteBuffer holds the paste buffer content
type PasteBuffer struct {
	content []byte
	mu      sync.RWMutex
}

var (
	globalPasteBuffer = &PasteBuffer{content: []byte{}}
)

// SetPasteBuffer sets the global paste buffer content
func SetPasteBuffer(content []byte) {
	globalPasteBuffer.mu.Lock()
	defer globalPasteBuffer.mu.Unlock()
	globalPasteBuffer.content = make([]byte, len(content))
	copy(globalPasteBuffer.content, content)
}

// GetPasteBuffer returns the global paste buffer content
func GetPasteBuffer() []byte {
	globalPasteBuffer.mu.RLock()
	defer globalPasteBuffer.mu.RUnlock()
	result := make([]byte, len(globalPasteBuffer.content))
	copy(result, globalPasteBuffer.content)
	return result
}

// EnterCopyMode enters copy mode for a window
func EnterCopyMode(win *session.Window, termFile *os.File, scrollback *ScrollbackBuffer) error {
	if scrollback == nil || scrollback.Size() == 0 {
		return fmt.Errorf("no scrollback available")
	}

	// Save terminal state
	oldState, err := term.MakeRaw(int(termFile.Fd()))
	if err != nil {
		return err
	}
	defer func() {
		_ = term.Restore(int(termFile.Fd()), oldState)
	}()

	// Initialize copy mode
	cm := &CopyMode{
		buffer:        scrollback,
		startLine:     scrollback.Size() - 1,
		startCol:      0,
		endLine:       scrollback.Size() - 1,
		endCol:        0,
		currentLine:   scrollback.Size() - 1,
		currentCol:    0,
		selecting:     false,
		selected:      false,
		searchMode:    false,
		searchTerm:    "",
		searchResults: make([]int, 0),
		searchIndex:   0,
	}

	// Enter copy mode loop
	return cm.run(termFile)
}

// run executes the copy mode interaction loop
func (cm *CopyMode) run(termFile *os.File) error {
	// Display copy mode prompt
	_, _ = fmt.Fprint(termFile, "\r\n[Copy mode - Use arrow keys to navigate, Space to mark, Enter to copy, / to search, q to quit]\r\n")

	buf := make([]byte, 1)
	searchInput := make([]byte, 0, 256)

	for {
		n, err := termFile.Read(buf)
		if err != nil || n == 0 {
			return err
		}

		key := buf[0]

		// Handle search mode
		if cm.searchMode {
			if key == '\r' || key == '\n' {
				// Execute search
				cm.executeSearch(string(searchInput))
				cm.searchMode = false
				searchInput = searchInput[:0]
				cm.updateDisplay(termFile)
				continue
			} else if key == 0x1b || key == 0x03 { // ESC or Ctrl+C
				// Cancel search
				cm.searchMode = false
				searchInput = searchInput[:0]
				cm.updateDisplay(termFile)
				continue
			} else if key == '\b' || key == 0x7f {
				// Backspace in search
				if len(searchInput) > 0 {
					searchInput = searchInput[:len(searchInput)-1]
					_, _ = fmt.Fprint(termFile, "\b \b")
				}
				continue
			} else if key >= 32 && key < 127 {
				// Add to search input
				searchInput = append(searchInput, key)
				_, _ = fmt.Fprint(termFile, string(key))
				continue
			}
		}

		// Handle escape sequences (arrow keys, etc.)
		if key == 0x1b { // ESC
			// Read more bytes for escape sequence
			seq := make([]byte, 0, 10)
			seq = append(seq, key)
			for i := 0; i < 10; i++ {
				b := make([]byte, 1)
				if n, _ := termFile.Read(b); n > 0 {
					seq = append(seq, b[0])
					if b[0] >= 0x40 && b[0] <= 0x7E {
						break
					}
				} else {
					break
				}
			}

			if len(seq) >= 3 && seq[1] == '[' {
				switch seq[2] {
				case 'A': // Up arrow
					cm.moveUp()
				case 'B': // Down arrow
					cm.moveDown()
				case 'C': // Right arrow
					cm.moveRight()
				case 'D': // Left arrow
					cm.moveLeft()
				}
			}
			continue
		}

		switch key {
		case 'q', 'Q':
			// Quit copy mode
			return nil
		case '/':
			// Enter search mode
			cm.searchMode = true
			searchInput = searchInput[:0]
			_, _ = fmt.Fprint(termFile, "\r\nSearch: ")
			continue
		case 'n', 'N':
			// Next search result
			if len(cm.searchResults) > 0 {
				cm.searchIndex = (cm.searchIndex + 1) % len(cm.searchResults)
				cm.currentLine = cm.searchResults[cm.searchIndex]
				cm.currentCol = 0
			}
		case ' ':
			// Mark start/end of selection
			cm.toggleMark()
		case '\r', '\n':
			// Copy selection and exit
			if cm.selected {
				cm.copySelection()
				return nil
			}
		case 'h', 'H':
			cm.moveLeft()
		case 'j', 'J':
			cm.moveDown()
		case 'k', 'K':
			cm.moveUp()
		case 'l', 'L':
			cm.moveRight()
		}

		// Update display
		cm.updateDisplay(termFile)
	}
}

// moveUp moves the cursor up one line
func (cm *CopyMode) moveUp() {
	if cm.currentLine > 0 {
		cm.currentLine--
		line := cm.buffer.GetLine(cm.currentLine)
		if cm.currentCol >= len(line) {
			cm.currentCol = len(line) - 1
			if cm.currentCol < 0 {
				cm.currentCol = 0
			}
		}
	}
}

// moveDown moves the cursor down one line
func (cm *CopyMode) moveDown() {
	if cm.currentLine < cm.buffer.Size()-1 {
		cm.currentLine++
		line := cm.buffer.GetLine(cm.currentLine)
		if cm.currentCol >= len(line) {
			cm.currentCol = len(line) - 1
			if cm.currentCol < 0 {
				cm.currentCol = 0
			}
		}
	}
}

// moveLeft moves the cursor left one column
func (cm *CopyMode) moveLeft() {
	if cm.currentCol > 0 {
		cm.currentCol--
	} else if cm.currentLine > 0 {
		cm.currentLine--
		line := cm.buffer.GetLine(cm.currentLine)
		cm.currentCol = len(line) - 1
		if cm.currentCol < 0 {
			cm.currentCol = 0
		}
	}
}

// moveRight moves the cursor right one column
func (cm *CopyMode) moveRight() {
	line := cm.buffer.GetLine(cm.currentLine)
	if cm.currentCol < len(line)-1 {
		cm.currentCol++
	} else if cm.currentLine < cm.buffer.Size()-1 {
		cm.currentLine++
		cm.currentCol = 0
	}
}

// toggleMark toggles the selection mark
func (cm *CopyMode) toggleMark() {
	if !cm.selecting {
		// Start selection
		cm.startLine = cm.currentLine
		cm.startCol = cm.currentCol
		cm.endLine = cm.currentLine
		cm.endCol = cm.currentCol
		cm.selecting = true
		cm.selected = false
	} else {
		// End selection
		cm.endLine = cm.currentLine
		cm.endCol = cm.currentCol
		cm.selecting = false
		cm.selected = true
	}
}

// copySelection copies the selected text to the paste buffer
func (cm *CopyMode) copySelection() {
	if !cm.selected {
		return
	}

	// Normalize selection (start should be before end)
	startLine := cm.startLine
	startCol := cm.startCol
	endLine := cm.endLine
	endCol := cm.endCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	// Collect selected text
	var selectedText []byte
	for line := startLine; line <= endLine; line++ {
		lineData := cm.buffer.GetLine(line)
		if line == startLine && line == endLine {
			// Single line selection
			if startCol < len(lineData) && endCol <= len(lineData) {
				selectedText = append(selectedText, lineData[startCol:endCol]...)
			}
		} else if line == startLine {
			// First line
			if startCol < len(lineData) {
				selectedText = append(selectedText, lineData[startCol:]...)
			}
			selectedText = append(selectedText, '\n')
		} else if line == endLine {
			// Last line
			if endCol <= len(lineData) {
				selectedText = append(selectedText, lineData[:endCol]...)
			}
		} else {
			// Middle line
			selectedText = append(selectedText, lineData...)
			selectedText = append(selectedText, '\n')
		}
	}

	// Set paste buffer
	SetPasteBuffer(selectedText)
}

// executeSearch searches for the term in scrollback
func (cm *CopyMode) executeSearch(term string) {
	cm.searchTerm = term
	cm.searchResults = make([]int, 0)

	if term == "" {
		return
	}

	// Search through all lines in scrollback
	for i := 0; i < cm.buffer.Size(); i++ {
		line := cm.buffer.GetLine(i)
		if len(line) > 0 {
			// Simple case-insensitive search
			lineLower := ""
			for _, r := range line {
				if r >= 'A' && r <= 'Z' {
					lineLower += string(r + 32)
				} else {
					lineLower += string(r)
				}
			}
			termLower := ""
			for _, r := range term {
				if r >= 'A' && r <= 'Z' {
					termLower += string(r + 32)
				} else {
					termLower += string(r)
				}
			}

			// Check if line contains search term
			for j := 0; j <= len(lineLower)-len(termLower); j++ {
				if lineLower[j:j+len(termLower)] == termLower {
					cm.searchResults = append(cm.searchResults, i)
					break
				}
			}
		}
	}

	// Move to first result if any found
	if len(cm.searchResults) > 0 {
		cm.searchIndex = 0
		cm.currentLine = cm.searchResults[0]
		cm.currentCol = 0
	}
}

// updateDisplay updates the copy mode display
func (cm *CopyMode) updateDisplay(termFile *os.File) {
	// Simple display - show current position
	line := cm.buffer.GetLine(cm.currentLine)
	lineStr := string(line)
	if cm.currentCol < len(lineStr) {
		lineStr = lineStr[:cm.currentCol] + "_" + lineStr[cm.currentCol:]
	} else {
		lineStr += "_"
	}

	status := fmt.Sprintf("\r[Line %d/%d, Col %d] %s",
		cm.currentLine+1, cm.buffer.Size(), cm.currentCol+1, lineStr)
	if len(status) > 80 {
		status = status[:77] + "..."
	}
	_, _ = fmt.Fprint(termFile, status)
}

// WritePasteBufferToFile writes the paste buffer to a file
func WritePasteBufferToFile(filename string) error {
	content := GetPasteBuffer()
	return os.WriteFile(filename, content, 0644)
}

// ReadPasteBufferFromFile reads the paste buffer from a file
func ReadPasteBufferFromFile(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	SetPasteBuffer(content)
	return nil
}

// WriteScrollbackToFile writes the scrollback buffer to a file
func WriteScrollbackToFile(scrollback *ScrollbackBuffer, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = scrollback.WriteTo(file)
	return err
}
