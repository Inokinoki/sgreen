package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/inoki/sgreen/internal/session"
)

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
	fmt.Fprint(out, helpText)
}

// ShowCommandPrompt displays a command prompt and executes commands
func ShowCommandPrompt(in, out *os.File, sess *session.Session, config *AttachConfig, scrollback *ScrollbackBuffer) error {
	fmt.Fprint(out, "\r\n: ")
	
	// Read command line
	cmdLine := make([]byte, 0, 256)
	buf := make([]byte, 1)
	
	for {
		n, err := in.Read(buf)
		if err != nil || n == 0 {
			return err
		}
		
		b := buf[0]
		if b == '\r' || b == '\n' {
			break
		}
		if b == '\b' || b == 0x7f {
			// Backspace
			if len(cmdLine) > 0 {
				cmdLine = cmdLine[:len(cmdLine)-1]
				fmt.Fprint(out, "\b \b")
			}
		} else if b >= 32 && b < 127 {
			// Printable character
			cmdLine = append(cmdLine, b)
			fmt.Fprint(out, string(b))
		}
	}
	
	cmd := string(cmdLine)
	if cmd == "" {
		return nil
	}
	
	// Parse and execute command
	return executeCommand(cmd, sess, config, scrollback)
}

// executeCommand executes a screen command
func executeCommand(cmd string, sess *session.Session, config *AttachConfig, scrollback *ScrollbackBuffer) error {
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
				win.GetPTYProcess().Pty.Write(pasteContent)
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
		
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

