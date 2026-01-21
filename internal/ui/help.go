package ui

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/inoki/sgreen/internal/pty"
	"github.com/inoki/sgreen/internal/session"
)

// Command history storage
var (
	commandHistory []string
	maxHistory     int = 100
)

// Available commands for completion
var availableCommands = []string{
	"title", "kill", "next", "prev", "select", "copy", "paste",
	"writebuf", "readbuf", "dump", "list", "help", "quit", "detach",
	"rename", "lock", "acladd", "acldel", "acl", "layout",
}

// ShowHelp displays the help screen with key bindings
func ShowHelp(out *os.File) {
	helpText := `
sgreen Key Bindings:

Window Management:
  C-a c          Create new window
  C-a n          Next window
  C-a p          Previous window
  C-a 0-9        Switch to window by number
  C-a A-Z        Switch to window A-Z
  C-a C-a        Toggle to last window
  C-a '          Select window by name/number
  C-a "          Show window list
  C-a k          Kill current window
  C-a A          Set window title

Scrollback and Copy/Paste:
  C-a [          Enter copy mode
  C-a ]          Paste from buffer
  C-a {          Write paste buffer to file
  C-a }          Read paste buffer from file
  C-a <          Dump scrollback to file
  C-a >          Write scrollback to file

Commands:
  C-a ?          Show this help
  C-a :          Command prompt
  C-a .          Redraw screen
  C-a d          Detach from session
  C-a a          Send literal C-a to program

Copy Mode (when in C-a [):
  Arrow keys     Navigate
  h/j/k/l        Navigate (vi-style)
  Space          Mark start/end of selection
  Enter          Copy selection and exit
  q              Quit copy mode

Command Prompt Commands:
  title <text>   Set window title
  kill           Kill current window
  next           Next window
  prev           Previous window
  select <n>     Switch to window n
  copy           Enter copy mode
  paste          Paste from buffer
  writebuf <f>   Write paste buffer to file
  readbuf <f>    Read paste buffer from file
  dump <f>       Dump scrollback to file

Press any key to continue...
`
	_, _ = fmt.Fprint(out, helpText)
}

// ShowCommandPrompt displays a command prompt and executes commands
func ShowCommandPrompt(in, out *os.File, sess *session.Session, config *AttachConfig, scrollback *ScrollbackBuffer) error {
	_, _ = fmt.Fprint(out, "\r\n: ")

	// Read command line with history and completion support
	cmdLine := make([]byte, 0, 256)
	buf := make([]byte, 1)
	currentHistoryIndex := -1
	originalCmd := ""

	for {
		n, err := in.Read(buf)
		if err != nil || n == 0 {
			return err
		}

		b := buf[0]

		// Handle escape sequences (arrow keys, etc.)
		if b == 0x1b { // ESC
			// Read next bytes to determine escape sequence
			seq := make([]byte, 0, 4)
			for i := 0; i < 3; i++ {
				n2, err2 := in.Read(buf)
				if err2 != nil || n2 == 0 {
					break
				}
				seq = append(seq, buf[0])
				if buf[0] >= 0x40 && buf[0] <= 0x7E {
					break
				}
			}

			// Handle arrow keys
			if len(seq) >= 2 && seq[0] == '[' {
				switch seq[1] {
				case 'A': // Up arrow - history previous
					if len(commandHistory) > 0 {
						if currentHistoryIndex == -1 {
							originalCmd = string(cmdLine)
							currentHistoryIndex = len(commandHistory) - 1
						} else if currentHistoryIndex > 0 {
							currentHistoryIndex--
						}
						// Clear current line
						_, _ = fmt.Fprintf(out, "\r\033[K: ")
						cmdLine = []byte(commandHistory[currentHistoryIndex])
						_, _ = fmt.Fprint(out, string(cmdLine))
					}
					continue
				case 'B': // Down arrow - history next
					if currentHistoryIndex >= 0 {
						if currentHistoryIndex < len(commandHistory)-1 {
							currentHistoryIndex++
							// Clear current line
							_, _ = fmt.Fprintf(out, "\r\033[K: ")
							cmdLine = []byte(commandHistory[currentHistoryIndex])
							_, _ = fmt.Fprint(out, string(cmdLine))
						} else {
							// Restore original command
							currentHistoryIndex = -1
							_, _ = fmt.Fprintf(out, "\r\033[K: ")
							cmdLine = []byte(originalCmd)
							_, _ = fmt.Fprint(out, string(cmdLine))
						}
					}
					continue
				case 'C': // Right arrow
					continue
				case 'D': // Left arrow
					continue
				}
			}
			continue
		}

		if b == '\r' || b == '\n' {
			break
		}

		if b == '\t' {
			// Tab completion
			currentCmd := string(cmdLine)
			matches := findCommandMatches(currentCmd)
			if len(matches) == 1 {
				// Single match - complete it
				completed := matches[0]
				// Clear and rewrite
				_, _ = fmt.Fprintf(out, "\r\033[K: ")
				cmdLine = []byte(completed + " ")
				_, _ = fmt.Fprint(out, string(cmdLine))
			} else if len(matches) > 1 {
				// Multiple matches - show them
				_, _ = fmt.Fprintf(out, "\r\n")
				for _, match := range matches {
					_, _ = fmt.Fprintf(out, "%s ", match)
				}
				_, _ = fmt.Fprintf(out, "\r\n: ")
				_, _ = fmt.Fprint(out, string(cmdLine))
			}
			continue
		}

		if b == '\b' || b == 0x7f {
			// Backspace
			if len(cmdLine) > 0 {
				cmdLine = cmdLine[:len(cmdLine)-1]
				_, _ = fmt.Fprintf(out, "\b \b")
			}
			currentHistoryIndex = -1 // Reset history navigation
		} else if b >= 32 && b < 127 {
			// Printable character
			cmdLine = append(cmdLine, b)
			_, _ = fmt.Fprint(out, string(b))
			currentHistoryIndex = -1 // Reset history navigation
		}
	}

	cmd := strings.TrimSpace(string(cmdLine))
	if cmd == "" {
		return nil
	}

	// Add to history (avoid duplicates)
	if len(commandHistory) == 0 || commandHistory[len(commandHistory)-1] != cmd {
		commandHistory = append(commandHistory, cmd)
		if len(commandHistory) > maxHistory {
			commandHistory = commandHistory[1:]
		}
	}
	// Parse and execute commands (support semicolon-separated commands)
	commands := strings.Split(cmd, ";")
	for _, singleCmd := range commands {
		singleCmd = strings.TrimSpace(singleCmd)
		if singleCmd == "" {
			continue
		}
		if err := executeCommand(singleCmd, sess, config, scrollback, in, out); err != nil {
			// Return error on first failure
			return err
		}
	}
	return nil
}

// findCommandMatches finds commands that match the prefix
func findCommandMatches(prefix string) []string {
	matches := make([]string, 0)
	prefixLower := strings.ToLower(prefix)

	for _, cmd := range availableCommands {
		if strings.HasPrefix(strings.ToLower(cmd), prefixLower) {
			matches = append(matches, cmd)
		}
	}

	return matches
}

func hasRedirectionTokens(args []string) bool {
	for _, arg := range args {
		if strings.Contains(arg, ">") || strings.Contains(arg, "<") {
			return true
		}
		if strings.HasPrefix(arg, "2>") || strings.HasPrefix(arg, "1>") || strings.HasPrefix(arg, "&>") {
			return true
		}
	}
	return false
}

// executeCommand executes a screen command
func executeCommand(cmd string, sess *session.Session, config *AttachConfig, scrollback *ScrollbackBuffer, in, out *os.File) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "title":
		if len(args) > 0 {
			title := strings.Join(args, " ")
			sess.SetWindowTitle(title)
		}
		return nil

	case "kill":
		return sess.KillCurrentWindow()

	case "next":
		sess.NextWindow()
		return nil

	case "prev":
		sess.PrevWindow()
		return nil

	case "select":
		if len(args) > 0 {
			return sess.SwitchToWindow(args[0])
		}
		return fmt.Errorf("usage: select <window>")

	case "copy":
		// Enter copy mode
		win := sess.GetCurrentWindow()
		if win == nil {
			return fmt.Errorf("no current window")
		}
		return EnterCopyMode(win, os.Stdin, scrollback)

	case "paste":
		// Paste from buffer
		pasteContent := GetPasteBuffer()
		if len(pasteContent) > 0 {
			win := sess.GetCurrentWindow()
			if win != nil && win.GetPTYProcess() != nil {
				if _, err := win.GetPTYProcess().Pty.Write(pasteContent); err != nil {
					return err
				}
			}
		}
		return nil

	case "writebuf":
		if len(args) > 0 {
			return WritePasteBufferToFile(args[0])
		}
		return fmt.Errorf("usage: writebuf <filename>")

	case "readbuf":
		if len(args) > 0 {
			return ReadPasteBufferFromFile(args[0])
		}
		return fmt.Errorf("usage: readbuf <filename>")

	case "dump":
		if len(args) > 0 {
			if scrollback == nil {
				return fmt.Errorf("no scrollback available")
			}
			return WriteScrollbackToFile(scrollback, args[0])
		}
		return fmt.Errorf("usage: dump <filename>")

	case "quit", "exit":
		// Exit all windows
		return fmt.Errorf("quit")

	case "rename":
		if len(args) > 0 {
			newName := args[0]
			if err := sess.Rename(newName); err != nil {
				return fmt.Errorf("failed to rename session: %w", err)
			}
			_, _ = fmt.Fprintf(out, "\r\nSession renamed to: %s\r\n", newName)
			return nil
		}
		return fmt.Errorf("usage: rename <new-name>")

	case "lock":
		// Lock screen (same as C-a x)
		// lockScreen is defined in attach.go (same package)
		return lockScreen(in, out)

	case "acladd":
		if len(args) == 0 {
			return fmt.Errorf("usage: acladd <user>")
		}
		user := args[0]
		if err := sess.AddUser(user); err != nil {
			return fmt.Errorf("failed to add user: %w", err)
		}
		_, _ = fmt.Fprintf(out, "\r\nAdded user: %s\r\n", user)
		return nil

	case "acldel":
		if len(args) == 0 {
			return fmt.Errorf("usage: acldel <user>")
		}
		user := args[0]
		if err := sess.RemoveUser(user); err != nil {
			return fmt.Errorf("failed to remove user: %w", err)
		}
		_, _ = fmt.Fprintf(out, "\r\nRemoved user: %s\r\n", user)
		return nil

	case "acl":
		_, _ = fmt.Fprintf(out, "\r\nOwner: %s\r\n", sess.Owner)
		if len(sess.AllowedUsers) == 0 {
			_, _ = fmt.Fprintf(out, "Allowed users: (none)\r\n")
		} else {
			_, _ = fmt.Fprintf(out, "Allowed users: %s\r\n", strings.Join(sess.AllowedUsers, ", "))
		}
		return nil

	case "screen":
		// Create window with command: screen [n] [cmd [args]]
		// Parse: screen [number] [command] [args...]
		if len(args) == 0 {
			// No args - create shell window
			shellPath := "/bin/sh"
			if envShell := os.Getenv("SHELL"); envShell != "" {
				shellPath = envShell
			}
			sessConfig := &session.Config{
				Term:            config.Term,
				UTF8:            config.UTF8,
				Encoding:        config.Encoding,
				AllCapabilities: config.AllCapabilities,
			}
			_, err := sess.CreateWindow(shellPath, []string{}, sessConfig)
			return err
		}

		// Check if first arg is a number (0-9)
		windowNum := -1
		cmdStart := 0
		if len(args) > 0 {
			if num, err := strconv.Atoi(args[0]); err == nil && num >= 0 && num <= 9 {
				windowNum = num
				cmdStart = 1
			}
		}

		// Get command and args
		if cmdStart >= len(args) {
			// No command specified - create shell
			shellPath := "/bin/sh"
			if envShell := os.Getenv("SHELL"); envShell != "" {
				shellPath = envShell
			}
			sessConfig := &session.Config{
				Term:            config.Term,
				UTF8:            config.UTF8,
				Encoding:        config.Encoding,
				AllCapabilities: config.AllCapabilities,
			}
			win, err := sess.CreateWindow(shellPath, []string{}, sessConfig)
			if err == nil && windowNum >= 0 {
				// Note: Setting specific window number would require renumbering
				// For now, window is created with next available number
				_ = win
			}
			return err
		}

		cmdPath := args[cmdStart]
		cmdArgs := args[cmdStart+1:]

		sessConfig := &session.Config{
			Term:            config.Term,
			UTF8:            config.UTF8,
			Encoding:        config.Encoding,
			AllCapabilities: config.AllCapabilities,
		}
		win, err := sess.CreateWindow(cmdPath, cmdArgs, sessConfig)
		if err == nil && windowNum >= 0 {
			// Note: Setting specific window number would require renumbering
			// For now, window is created with next available number
			_ = win
		}
		return err

	case "exec":
		// Execute command in current window: exec [cmd [args]]
		if len(args) == 0 {
			return fmt.Errorf("usage: exec <command> [args...]")
		}

		cmdPath := args[0]
		cmdArgs := args[1:]
		// If redirection patterns are present, execute via shell.
		if hasRedirectionTokens(args) {
			cmdLine := strings.Join(args, " ")
			if runtime.GOOS == "windows" {
				cmdPath = "cmd"
				cmdArgs = []string{"/C", cmdLine}
			} else {
				shellPath := "/bin/sh"
				if envShell := os.Getenv("SHELL"); envShell != "" {
					shellPath = envShell
				}
				cmdPath = shellPath
				cmdArgs = []string{"-c", cmdLine}
			}
		}

		// Get current window
		win := sess.GetCurrentWindow()
		if win == nil {
			return fmt.Errorf("no current window")
		}

		// Kill current process in window
		if ptyProc := win.GetPTYProcess(); ptyProc != nil {
			if ptyProc.Cmd != nil && ptyProc.Cmd.Process != nil {
				if err := ptyProc.Cmd.Process.Kill(); err != nil {
					return err
				}
			}
		}

		// Start new process
		sessConfig := &session.Config{
			Term:            config.Term,
			UTF8:            config.UTF8,
			Encoding:        config.Encoding,
			AllCapabilities: config.AllCapabilities,
		}

		// Create new PTY process
		ptyProc, err := pty.StartWithEnv(cmdPath, cmdArgs, map[string]string{
			"TERM": sessConfig.Term,
		})
		if err != nil {
			return fmt.Errorf("failed to exec command: %w", err)
		}

		// Update window with new PTY process
		win.SetPTYProcess(ptyProc)
		win.Pid = ptyProc.Cmd.Process.Pid
		win.CmdPath = cmdPath
		win.CmdArgs = cmdArgs

		return nil

	case "layout":
		if len(args) == 0 {
			return fmt.Errorf("usage: layout <save|select|list> [name]")
		}
		switch args[0] {
		case "save":
			if len(args) < 2 {
				return fmt.Errorf("usage: layout save <name>")
			}
			if err := sess.SaveLayout(args[1]); err != nil {
				return fmt.Errorf("failed to save layout: %w", err)
			}
			_, _ = fmt.Fprintf(out, "\r\nSaved layout: %s\r\n", args[1])
			return nil
		case "select":
			if len(args) < 2 {
				return fmt.Errorf("usage: layout select <name>")
			}
			if err := sess.SelectLayout(args[1]); err != nil {
				return fmt.Errorf("failed to select layout: %w", err)
			}
			_, _ = fmt.Fprintf(out, "\r\nSelected layout: %s\r\n", args[1])
			return nil
		case "list":
			names := sess.ListLayouts()
			if len(names) == 0 {
				_, _ = fmt.Fprintf(out, "\r\nNo layouts saved\r\n")
				return nil
			}
			_, _ = fmt.Fprintf(out, "\r\nLayouts: %s\r\n", strings.Join(names, ", "))
			return nil
		default:
			return fmt.Errorf("usage: layout <save|select|list> [name]")
		}

	case "displays":
		// List displays (multi-user sessions)
		// For now, just show current session info
		_, _ = fmt.Fprintf(out, "\r\nSession: %s\r\n", sess.ID)
		_, _ = fmt.Fprintf(out, "Windows: %d\r\n", len(sess.Windows))
		for i, win := range sess.Windows {
			_, _ = fmt.Fprintf(out, "  Window %d: %s (PID: %d)\r\n", i, win.Title, win.Pid)
		}
		return nil

	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}
