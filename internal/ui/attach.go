//go:build !windows
// +build !windows

package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/inoki/sgreen/internal/session"
)

var (
	// ErrDetach is returned when the user detaches from a session
	ErrDetach = errors.New("detached from session")
)

// ErrWindowCommand is returned when a window command is detected
type ErrWindowCommand struct {
	Command string
	Window  string
	Title   string
}

func (e *ErrWindowCommand) Error() string {
	return fmt.Sprintf("window command: %s", e.Command)
}

// Attach attaches the current terminal to a session
func Attach(in *os.File, out *os.File, errOut *os.File, sess *session.Session) error {
	return AttachWithConfig(in, out, errOut, sess, DefaultAttachConfig())
}

// AttachWithConfig attaches the current terminal to a session with configuration
func AttachWithConfig(in *os.File, out *os.File, errOut *os.File, sess *session.Session, config *AttachConfig) error {
	// Get PTY process
	ptyProc := sess.GetPTYProcess()
	if ptyProc == nil {
		return errors.New("PTY process not available")
	}

	// Save original terminal state
	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(in.Fd()), oldState)

	// Main attach loop - handles window switching
	return attachLoop(in, out, errOut, sess, config)
}

// attachLoop is the main loop that handles window switching
func attachLoop(in *os.File, out *os.File, errOut *os.File, sess *session.Session, config *AttachConfig) error {
	// Handle window size changes (Unix only)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, unix.SIGWINCH)
	defer signal.Stop(sigChan)

	// Create scrollback buffers for windows (stored in a map)
	scrollbackBuffers := make(map[int]*ScrollbackBuffer)

	for {
		// Get current window
		win := sess.GetCurrentWindow()
		if win == nil {
			return fmt.Errorf("no current window")
		}
		
		ptyProc := win.GetPTYProcess()
		if ptyProc == nil {
			return fmt.Errorf("current window has no PTY process")
		}

		// Get or create scrollback buffer for this window
		scrollback, exists := scrollbackBuffers[win.ID]
		if !exists {
			scrollbackSize := 1000 // Default
			if win.ScrollbackSize > 0 {
				scrollbackSize = win.ScrollbackSize
			} else if config.Scrollback > 0 {
				scrollbackSize = config.Scrollback
			}
			scrollback = NewScrollbackBuffer(scrollbackSize)
			scrollbackBuffers[win.ID] = scrollback
		}

		// Create output writer (with logging if enabled)
		outputWriter := createOutputWriter(out, config)

		// Wrap output writer to also write to scrollback
		scrollbackWriter := io.MultiWriter(outputWriter, &scrollbackWriter{scrollback: scrollback})

		// Apply output optimization if requested
		if config.OptimalOutput {
			scrollbackWriter = createOptimalWriter(scrollbackWriter)
		}

		// Handle flow control
		flowControl := setupFlowControl(config.FlowControl, config.Interrupt)

		// Set window size
		if err := setWindowSizeForWindow(in, win, config.AdaptSize); err != nil {
			// Non-fatal
		}

		// Monitor window size changes
		go func() {
			for range sigChan {
				if win := sess.GetCurrentWindow(); win != nil {
					setWindowSizeForWindow(in, win, config.AdaptSize)
				}
			}
		}()

		// Copy from PTY to output with flow control
		outputDone := make(chan error, 1)
		go func() {
			outputDone <- copyWithFlowControl(ptyProc.Pty, scrollbackWriter, flowControl)
		}()

		// Create a reader that detects detach sequence and window commands
		detachReader := newDetachReaderWithConfig(in, config)

		// Copy from input to PTY, with detach detection and window commands
		inputDone := make(chan error, 1)
		go func() {
			_, err := io.Copy(ptyProc.Pty, detachReader)
			inputDone <- err
		}()

		// Wait for either input or output to finish
		select {
		case err := <-inputDone:
			if err == ErrDetach {
				// User detached, this is normal
				return nil
			}
			
			// Check if it's a window command
			var winCmd *ErrWindowCommand
			if errors.As(err, &winCmd) {
				// Get current scrollback for command handling
				currentScrollback := scrollbackBuffers[win.ID]
				if handleErr := handleWindowCommand(sess, winCmd, config, in, out, currentScrollback); handleErr != nil {
					// If command handling fails, return error
					return handleErr
				}
				// Window switched, restart the loop
				continue
			}
			
			// Other error
			return err
			
		case err := <-outputDone:
			// Output finished (EOF or error)
			if err == io.EOF {
				// PTY closed, try to continue with next window or exit
				return nil
			}
			return err
		}
	}
}

// setWindowSizeForWindow sets the PTY window size for a specific window
func setWindowSizeForWindow(termFile *os.File, win *session.Window, adaptSize bool) error {
	width, height, err := term.GetSize(int(termFile.Fd()))
	if err != nil {
		return err
	}

	ptyProc := win.GetPTYProcess()
	if ptyProc == nil {
		return errors.New("PTY process not available")
	}

	return ptyProc.SetSize(uint16(height), uint16(width))
}

// scrollbackWriter wraps a writer to also write to scrollback buffer
type scrollbackWriter struct {
	scrollback *ScrollbackBuffer
}

func (sw *scrollbackWriter) Write(p []byte) (n int, err error) {
	sw.scrollback.AppendBytes(p)
	return len(p), nil
}

// handleWindowCommand handles window management commands
func handleWindowCommand(sess *session.Session, cmd *ErrWindowCommand, config *AttachConfig, in, out *os.File, scrollback *ScrollbackBuffer) error {
	switch cmd.Command {
	case "create":
		// Create new window with default shell
		shellPath := "/bin/sh"
		if envShell := os.Getenv("SHELL"); envShell != "" {
			shellPath = envShell
		}
		
		sessConfig := &session.Config{
			Term: config.Term,
			UTF8: false, // TODO: get from config
			AllCapabilities: config.AllCapabilities,
		}
		
		_, err := sess.CreateWindow(shellPath, []string{}, sessConfig)
		if err != nil {
			return fmt.Errorf("failed to create window: %w", err)
		}
		return nil
		
	case "next":
		sess.NextWindow()
		return nil
		
	case "prev":
		sess.PrevWindow()
		return nil
		
	case "toggle":
		sess.ToggleLastWindow()
		return nil
		
	case "switch":
		if cmd.Window == "" {
			return fmt.Errorf("no window specified")
		}
		return sess.SwitchToWindow(cmd.Window)
		
	case "kill":
		return sess.KillCurrentWindow()
		
	case "title":
		sess.SetWindowTitle(cmd.Title)
		return nil
		
	case "list":
		// Show window list - for now, just switch (full implementation would show interactive list)
		// This is a placeholder - full implementation would show a selectable list
		return nil
		
	case "copymode":
		// Enter copy mode
		win := sess.GetCurrentWindow()
		if win == nil {
			return fmt.Errorf("no current window")
		}
		return EnterCopyMode(win, in, scrollback)
		
	case "paste":
		// Paste from buffer
		pasteContent := GetPasteBuffer()
		if len(pasteContent) > 0 {
			win := sess.GetCurrentWindow()
			if win != nil && win.GetPTYProcess() != nil {
				win.GetPTYProcess().Pty.Write(pasteContent)
			}
		}
		return nil
		
	case "writebuffer":
		// Write paste buffer to file
		if cmd.Title == "" {
			return fmt.Errorf("no filename specified")
		}
		return WritePasteBufferToFile(cmd.Title)
		
	case "readbuffer":
		// Read paste buffer from file
		if cmd.Title == "" {
			return fmt.Errorf("no filename specified")
		}
		return ReadPasteBufferFromFile(cmd.Title)
		
		case "dumpscrollback":
			// Dump scrollback to file
			if cmd.Title == "" {
				return fmt.Errorf("no filename specified")
			}
			if scrollback == nil {
				return fmt.Errorf("no scrollback available")
			}
			return WriteScrollbackToFile(scrollback, cmd.Title)
		
		case "help":
			// Show help
			ShowHelp(out)
			// Wait for key press
			buf := make([]byte, 1)
			in.Read(buf)
			return nil
		
		case "command":
			// Show command prompt
			return ShowCommandPrompt(in, out, sess, config, scrollback)
		
		case "redraw":
			// Redraw screen - clear and redraw
			fmt.Fprint(out, "\033[2J\033[H")
			return nil
		
		default:
			return fmt.Errorf("unknown window command: %s", cmd.Command)
	}
}

// getShellPath returns the default shell path
func getShellPath() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}

// createOptimalWriter creates an optimized output writer
func createOptimalWriter(w io.Writer) io.Writer {
	// For optimal output, we can add buffering or other optimizations
	// For now, return a buffered writer
	return w
}

// FlowControlConfig holds flow control configuration
type FlowControlConfig struct {
	Enabled   bool
	Auto      bool
	Interrupt bool
}

// setupFlowControl sets up flow control based on configuration
func setupFlowControl(flowControl string, interrupt bool) *FlowControlConfig {
	cfg := &FlowControlConfig{
		Enabled:   false,
		Auto:      false,
		Interrupt: interrupt,
	}

	switch flowControl {
	case "on":
		cfg.Enabled = true
	case "off":
		cfg.Enabled = false
	case "auto":
		cfg.Enabled = true
		cfg.Auto = true
	default:
		// Default: off
		cfg.Enabled = false
	}

	return cfg
}

// copyWithFlowControl copies data with flow control handling
func copyWithFlowControl(src io.Reader, dst io.Writer, flowControl *FlowControlConfig) error {
	// Basic implementation - in full version would handle XON/XOFF
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				// Handle write errors (flow control)
				if flowControl.Enabled && flowControl.Interrupt {
					// Interrupt immediately on flow control
					return writeErr
				}
			}
		}
		if err != nil {
			return err
		}
	}
}

// setWindowSize sets the PTY window size to match the terminal (backward compatibility)
func setWindowSize(termFile *os.File, sess *session.Session, adaptSize bool) error {
	win := sess.GetCurrentWindow()
	if win == nil {
		return errors.New("no current window")
	}
	return setWindowSizeForWindow(termFile, win, adaptSize)
}

// detachReader wraps an io.Reader to detect the detach sequence
type detachReader struct {
	reader      io.Reader
	state       int    // 0: normal, 1: saw command char
	pending     []byte // bytes to output before reading more
	commandChar byte   // Command character (default: Ctrl+A = 0x01)
	literalChar byte   // Literal escape character (default: 'a')
}

func newDetachReader(reader io.Reader) *detachReader {
	return newDetachReaderWithConfig(reader, DefaultAttachConfig())
}

func newDetachReaderWithConfig(reader io.Reader, config *AttachConfig) *detachReader {
	return &detachReader{
		reader:      reader,
		state:       0,
		pending:     make([]byte, 0, 2),
		commandChar: config.CommandChar,
		literalChar: config.LiteralChar,
	}
}

func (dr *detachReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// First, output any pending bytes
	if len(dr.pending) > 0 {
		copied := copy(p, dr.pending)
		dr.pending = dr.pending[copied:]
		if copied > 0 {
			return copied, nil
		}
	}

	// Read one byte at a time to detect escape sequences
	buf := make([]byte, 1)
	read, err := dr.reader.Read(buf)
	if err != nil {
		return 0, err
	}

	if read == 0 {
		return 0, nil
	}

	b := buf[0]

	switch dr.state {
	case 0:
		// Normal state
		if b == dr.commandChar {
			dr.state = 1
			// Don't output command char, wait for next character
			return 0, nil
		}
		// Normal byte
		p[0] = b
		return 1, nil

	case 1:
		// Saw command char, waiting for command
		switch b {
		case 'd':
			// Detach sequence detected
			return 0, ErrDetach
		case dr.literalChar:
			// Literal command char - send the command char to the program
			p[0] = dr.commandChar
			dr.state = 0
			return 1, nil
		case 'a':
			// C-a a: Send literal C-a to program (alternative to literal char)
			p[0] = dr.commandChar
			dr.state = 0
			return 1, nil
		case dr.commandChar:
			// C-a C-a: Toggle to last window
			return 0, &ErrWindowCommand{Command: "toggle"}
		case 'c':
			// Create new window - handled by command handler
			return 0, &ErrWindowCommand{Command: "create"}
		case 'n':
			// Next window
			return 0, &ErrWindowCommand{Command: "next"}
		case 'p':
			// Previous window
			return 0, &ErrWindowCommand{Command: "prev"}
		case 'k':
			// Kill current window
			return 0, &ErrWindowCommand{Command: "kill"}
		case 'A':
			// Set window title - need to read title
			dr.state = 2 // Enter title input mode
			return 0, nil
		case '[':
			// Enter copy mode
			return 0, &ErrWindowCommand{Command: "copymode"}
		case ']':
			// Paste from buffer
			return 0, &ErrWindowCommand{Command: "paste"}
		case '{':
			// Write paste buffer to file
			dr.state = 4 // Enter filename input mode
			return 0, nil
		case '}':
			// Read paste buffer from file
			dr.state = 5 // Enter filename input mode
			return 0, nil
		case '<':
			// Dump scrollback to file
			dr.state = 6 // Enter filename input mode
			return 0, nil
		case '>':
			// Write scrollback to file
			dr.state = 7 // Enter filename input mode
			return 0, nil
		case '?':
			// Show help
			return 0, &ErrWindowCommand{Command: "help"}
		case ':':
			// Command prompt
			return 0, &ErrWindowCommand{Command: "command"}
		case '.':
			// Redraw screen
			return 0, &ErrWindowCommand{Command: "redraw"}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			// Switch to window 0-9
			return 0, &ErrWindowCommand{Command: "switch", Window: string(b)}
		case ' ':
			// Space: Next window (alternative)
			return 0, &ErrWindowCommand{Command: "next"}
		case '\b', 0x7f: // Backspace
			// Backspace: Previous window (alternative)
			return 0, &ErrWindowCommand{Command: "prev"}
		case '"':
			// Interactive window list - for now, just show list
			return 0, &ErrWindowCommand{Command: "list"}
		case '\'':
			// Select window by name/number - enter selection mode
			dr.state = 3 // Enter window selection mode
			return 0, nil
		default:
			// Check for A-Z (windows 10-35)
			if b >= 'A' && b <= 'Z' {
				return 0, &ErrWindowCommand{Command: "switch", Window: string(b)}
			}
			// Not a recognized command, output the command char we held back, then this byte
			dr.state = 0
			if len(p) >= 2 {
				p[0] = dr.commandChar
				p[1] = b
				return 2, nil
			}
			// Buffer too small, output command char and buffer the next byte
			p[0] = dr.commandChar
			dr.pending = append(dr.pending, b)
			return 1, nil
		}
	case 3:
		// Window selection mode - read until newline
		if b == '\n' || b == '\r' {
			dr.state = 0
			// Window number is in dr.pending
			windowNum := string(dr.pending)
			dr.pending = dr.pending[:0]
			return 0, &ErrWindowCommand{Command: "switch", Window: windowNum}
		}
		dr.pending = append(dr.pending, b)
		return 0, nil
	case 4:
		// Filename input mode for write buffer
		if b == '\n' || b == '\r' {
			dr.state = 0
			filename := string(dr.pending)
			dr.pending = dr.pending[:0]
			return 0, &ErrWindowCommand{Command: "writebuffer", Title: filename}
		}
		dr.pending = append(dr.pending, b)
		return 0, nil
	case 5:
		// Filename input mode for read buffer
		if b == '\n' || b == '\r' {
			dr.state = 0
			filename := string(dr.pending)
			dr.pending = dr.pending[:0]
			return 0, &ErrWindowCommand{Command: "readbuffer", Title: filename}
		}
		dr.pending = append(dr.pending, b)
		return 0, nil
	case 6:
		// Filename input mode for dump scrollback
		if b == '\n' || b == '\r' {
			dr.state = 0
			filename := string(dr.pending)
			dr.pending = dr.pending[:0]
			return 0, &ErrWindowCommand{Command: "dumpscrollback", Title: filename}
		}
		dr.pending = append(dr.pending, b)
		return 0, nil
	case 7:
		// Filename input mode for write scrollback
		if b == '\n' || b == '\r' {
			dr.state = 0
			filename := string(dr.pending)
			dr.pending = dr.pending[:0]
			return 0, &ErrWindowCommand{Command: "dumpscrollback", Title: filename}
		}
		dr.pending = append(dr.pending, b)
		return 0, nil
	case 2:
		// Title input mode - read until newline
		if b == '\n' || b == '\r' {
			dr.state = 0
			// Title is in dr.pending
			title := string(dr.pending)
			dr.pending = dr.pending[:0]
			return 0, &ErrWindowCommand{Command: "title", Title: title}
		}
		dr.pending = append(dr.pending, b)
		return 0, nil
	}

	return 0, nil
}

// createOutputWriter creates an output writer with optional logging
func createOutputWriter(out io.Writer, config *AttachConfig) io.Writer {
	if !config.Logging && config.Logfile == "" {
		return out
	}

	// Create multi-writer for both output and log file
	writers := []io.Writer{out}

	if config.Logfile != "" {
		logFile, err := os.OpenFile(config.Logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			writers = append(writers, logFile)
		}
	}

	return io.MultiWriter(writers...)
}

