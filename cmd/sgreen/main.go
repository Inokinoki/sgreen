package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/inoki/sgreen/internal/session"
	"github.com/inoki/sgreen/internal/ui"
)

// Config holds configuration options from command-line flags
type Config struct {
	Shell          string
	Term           string
	UTF8           bool
	AllCapabilities bool
	AdaptSize      bool
	Quiet          bool
	Logging        bool
	Logfile        string
	Scrollback     int
	CommandChar    string
	LiteralChar    string
	ConfigFile     string
	IgnoreSTY      bool
	OptimalOutput  bool
	PreselectWindow string
	Wipe           bool
	Version        bool
	SendCommand    string
	Multiuser      bool
	FlowControl    string // "on", "off", "auto"
	Interrupt      bool
}

func main() {
	// Parse flags
	var (
		reattach         = flag.Bool("r", false, "Reattach to a detached session")
		reattachOrCreate = flag.Bool("R", false, "Reattach or create if none exists")
		reattachOrCreateRR = flag.Bool("RR", false, "Reattach or create, detaching elsewhere if needed")
		powerDetach      = flag.Bool("D", false, "Power detach (force detach from elsewhere)")
		detach           = flag.Bool("d", false, "Detach a session")
		list             = flag.Bool("ls", false, "List all sessions")
		listAlt          = flag.Bool("list", false, "List all sessions (alternative)")
		sessionName      = flag.String("S", "", "Name the session")
		help             = flag.Bool("h", false, "Show help")
		helpLong         = flag.Bool("help", false, "Show help")
		
		// Session Configuration
		shell            = flag.String("s", "", "Shell program (default: /bin/sh or $SHELL)")
		configFile       = flag.String("c", "", "Config file instead of default .screenrc")
		escapeChars      = flag.String("e", "", "Command character and literal escape (default: ^Aa)")
		term             = flag.String("T", "", "Set TERM environment variable")
		utf8             = flag.Bool("U", false, "UTF-8 mode")
		allCapabilities  = flag.Bool("a", false, "Include all capabilities in termcap")
		adaptSize        = flag.Bool("A", false, "Adapt window sizes to new terminal size on attach")
		
		// Output and Logging
		logging          = flag.Bool("L", false, "Turn on output logging for windows")
		logfile          = flag.String("Logfile", "", "Log output to file")
		scrollback       = flag.Int("H", 0, "Set scrollback buffer size (screen uses -h, but conflicts with help)")
		
		// Other Options
		version          = flag.Bool("v", false, "Print version information")
		wipe             = flag.Bool("wipe", false, "Remove dead sessions from list")
		sendCommand      = flag.String("X", "", "Send command to a running session")
		ignoreSTY        = flag.Bool("m", false, "Ignore $STY environment variable")
		optimalOutput    = flag.Bool("O", false, "Use optimal output mode")
		preselectWindow  = flag.String("p", "", "Preselect a window")
		quiet            = flag.Bool("q", false, "Quiet startup (suppress messages)")
		interrupt        = flag.Bool("i", false, "Interrupt output immediately when flow control is on")
		flowControl      = flag.String("f", "", "Flow control: on, off, or auto")
		flowControlOff   = flag.Bool("fn", false, "Flow control off")
		flowControlAuto  = flag.Bool("fa", false, "Flow control automatic")
		multiuser        = flag.Bool("x", false, "Attach to a session without detaching it (multiuser)")
	)

	flag.Usage = printUsage
	flag.Parse()

	// Build config from flags
	config := &Config{
		Shell:          *shell,
		Term:           *term,
		UTF8:           *utf8,
		AllCapabilities: *allCapabilities,
		AdaptSize:      *adaptSize,
		Quiet:          *quiet,
		Logging:        *logging,
		Logfile:        *logfile,
		Scrollback:     *scrollback,
		CommandChar:    "",
		LiteralChar:    "",
		ConfigFile:     *configFile,
		IgnoreSTY:      *ignoreSTY,
		OptimalOutput:  *optimalOutput,
		PreselectWindow: *preselectWindow,
		Wipe:           *wipe,
		Version:        *version,
		SendCommand:    *sendCommand,
		Multiuser:      *multiuser,
		FlowControl:    *flowControl,
		Interrupt:      *interrupt,
	}
	
	// Handle -fn and -fa flags (screen-compatible)
	// These take precedence over -f value
	if *flowControlOff {
		config.FlowControl = "off"
	} else if *flowControlAuto {
		config.FlowControl = "auto"
	} else if *flowControl == "" {
		// If -f is not set and neither -fn nor -fa is set, check if -f flag was used
		// For screen compatibility: -f alone means "on"
		// But we use string flag, so empty means not set
	}

	// Parse escape characters
	if *escapeChars != "" {
		parts := strings.SplitN(*escapeChars, "", 2)
		if len(parts) >= 1 {
			config.CommandChar = parts[0]
		}
		if len(parts) >= 2 {
			config.LiteralChar = parts[1]
		}
	}

	// Handle version
	if *version {
		printVersion()
		return
	}

	// Handle help
	if *help || *helpLong {
		printUsage()
		return
	}

	// Handle wipe
	if *wipe {
		handleWipe()
		return
	}

	// Handle send command (-X)
	if *sendCommand != "" {
		handleSendCommand(*sessionName, *sendCommand)
		return
	}

	// Handle list
	if *list || *listAlt {
		handleList()
		return
	}

	// Handle power detach (-D)
	if *powerDetach {
		handlePowerDetach(*sessionName, flag.Args(), config)
		return
	}

	// Handle detach
	if *detach {
		handleDetach(*reattach, *sessionName)
		return
	}

	// Handle reattach or create with RR (-RR)
	if *reattachOrCreateRR {
		handleReattachOrCreateRR(*sessionName, flag.Args(), config)
		return
	}

	// Handle reattach or create (-R)
	if *reattachOrCreate {
		handleReattachOrCreate(*sessionName, flag.Args(), config)
		return
	}

	// Handle reattach
	if *reattach {
		// Check for multiuser mode (-x)
		if *multiuser {
			// Multiuser attach: don't detach from elsewhere
			config.Multiuser = true
		}
		handleReattachWithConfig(*sessionName, config)
		return
	}
	
	// Handle multiuser attach (-x)
	if *multiuser {
		config.Multiuser = true
		handleReattachWithConfig(*sessionName, config)
		return
	}

	// Default: create new session
	handleNew(*sessionName, flag.Args(), config)
}

// printVersion prints version information
func printVersion() {
	fmt.Println("sgreen version 0.1.0")
	fmt.Println("A simplified screen-like terminal multiplexer")
	fmt.Println("Compatible with GNU screen command-line interface")
}

// handleWipe removes dead sessions from the list
func handleWipe() {
	sessions := session.List()
	removed := 0
	
	for _, sess := range sessions {
		// Check if session is dead
		if !isProcessAliveByPID(sess.Pid) {
			// Remove dead session
			if err := session.Delete(sess.ID); err == nil {
				removed++
			}
		}
	}
	
	if removed > 0 {
		fmt.Printf("Removed %d dead session(s)\n", removed)
	} else {
		fmt.Println("No dead sessions found")
	}
}

// handleSendCommand sends a command to a running session
func handleSendCommand(sessionName, command string) {
	var sess *session.Session
	var err error
	
	if sessionName != "" {
		sess, err = session.Load(sessionName)
	} else {
		sessions := session.List()
		if len(sessions) == 0 {
			fmt.Fprintf(os.Stderr, "No screen session found.\n")
		os.Exit(1)
		}
		sess = sessions[0]
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "No screen session found: %s\n", sessionName)
		os.Exit(1)
	}

	// Execute command in session
	if err := session.ExecuteCommand(sess, command); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

// isQuiet checks if quiet mode is enabled
func isQuiet() bool {
	// Check for -q flag (would need to pass config around)
	return false
}

// handleNew creates a new session
func handleNew(sessionName string, cmdArgs []string, config *Config) {
	// Generate session name if not provided
	if sessionName == "" {
		// Use default name based on PID and timestamp
		sessionName = fmt.Sprintf("%d.%d", os.Getpid(), time.Now().Unix())
	}

	// Determine shell
	shellPath := "/bin/sh"
	if config.Shell != "" {
		shellPath = config.Shell
	} else if envShell := os.Getenv("SHELL"); envShell != "" {
		shellPath = envShell
	}

	// Default command is shell
	cmdPath := shellPath
	args := cmdArgs

	if len(cmdArgs) > 0 {
		cmdPath = cmdArgs[0]
		args = cmdArgs[1:]
	}

	// Check STY environment variable unless -m flag is set
	if !config.IgnoreSTY {
		if sty := os.Getenv("STY"); sty != "" {
			// STY format: pid.tty.host
			parts := strings.Split(sty, ".")
			if len(parts) >= 1 {
				// Try to use session from STY
				sess, err := session.Load(parts[0])
				if err == nil && sess != nil {
					if !config.Quiet {
						fmt.Fprintf(os.Stderr, "Attaching to session from $STY: %s\n", sess.ID)
					}
					attachToSession(sess, config)
					return
				}
			}
		}
	}

	// Load config file if specified
	if config.ConfigFile != "" {
		loadConfigFile(config.ConfigFile, config)
	}

	// Handle window preselection (-p)
	// Note: This is a placeholder for when multiple windows are implemented
	if config.PreselectWindow != "" {
		if !config.Quiet {
			fmt.Fprintf(os.Stderr, "Note: Window preselection (-p) requires multiple windows feature (not yet implemented)\n")
		}
	}

	// Check if session already exists
	existingSess, err := session.Load(sessionName)
	if err == nil && existingSess != nil {
		// Session exists, try to attach to it instead
		if !config.Quiet {
			fmt.Fprintf(os.Stderr, "Session %s already exists. Attaching...\n", sessionName)
		}
		attachToSession(existingSess, config)
		return
	}

	// Create new session with config
	sessConfig := &session.Config{
		Term:           config.Term,
		UTF8:           config.UTF8,
		Scrollback:     config.Scrollback,
		AllCapabilities: config.AllCapabilities,
	}
	sess, err := session.NewWithConfig(sessionName, cmdPath, args, sessConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}

	// Attach to the new session
	attachToSession(sess, config)
}

// handleReattach reattaches to an existing session
func handleReattach(sessionName string) {
	handleReattachWithConfig(sessionName, &Config{})
}

// handleReattachWithConfig reattaches with configuration
func handleReattachWithConfig(sessionName string, config *Config) {
	sessions := session.List()

	if len(sessions) == 0 {
		fmt.Fprintf(os.Stderr, "No screen session found.\n")
		os.Exit(1)
	}

	var sess *session.Session
	var err error

	if sessionName != "" {
		// Load specific session by name
		sess, err = session.Load(sessionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No screen session found: %s\n", sessionName)
			os.Exit(1)
		}
	} else {
		// Find first detached session, or first session if only one
		detached := findDetachedSessions(sessions)
		if len(detached) == 1 {
			sess = detached[0]
		} else if len(sessions) == 1 {
			sess = sessions[0]
		} else if len(detached) > 1 {
			fmt.Fprintf(os.Stderr, "There are several detached sessions:\n")
			printSessionList(sessions)
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, "No detached screen session found.\n")
			os.Exit(1)
		}
	}

	attachToSession(sess, config)
}

// handleReattachOrCreate implements -R flag: reattach or create if none exists
func handleReattachOrCreate(sessionName string, cmdArgs []string, config *Config) {
	sessions := session.List()

	// If no sessions exist, create a new one
	if len(sessions) == 0 {
		handleNew(sessionName, cmdArgs, config)
		return
	}

	// Try to find a session to reattach to
	var sess *session.Session
	var err error

	if sessionName != "" {
		// Try to load specific session
		sess, err = session.Load(sessionName)
		if err != nil {
			// Session not found, create new one with this name
			handleNew(sessionName, cmdArgs, config)
			return
		}
	} else {
		// Find first detached session, or first session if only one
		detached := findDetachedSessions(sessions)
		if len(detached) > 0 {
			sess = detached[0]
		} else if len(sessions) == 1 {
			sess = sessions[0]
		} else {
			// Multiple sessions, use first one
			sess = sessions[0]
		}
	}

	// Reattach to the found session
	attachToSession(sess, config)
}

// handleReattachOrCreateRR implements -RR flag: reattach or create, detaching elsewhere if needed
func handleReattachOrCreateRR(sessionName string, cmdArgs []string, config *Config) {
	sessions := session.List()

	// If no sessions exist, create a new one
	if len(sessions) == 0 {
		handleNew(sessionName, cmdArgs, config)
		return
	}

	var sess *session.Session
	var err error

	if sessionName != "" {
		sess, err = session.Load(sessionName)
		if err != nil {
			// Session not found, create new one
			handleNew(sessionName, cmdArgs, config)
			return
		}
	} else {
		// Find first session (prefer detached)
		detached := findDetachedSessions(sessions)
		if len(detached) > 0 {
			sess = detached[0]
		} else {
			sess = sessions[0]
		}
	}

	// Force detach if attached elsewhere, then attach
	if sess.GetPTYProcess() != nil {
		sess.ForceDetach()
	}

	attachToSession(sess, config)
}

// handlePowerDetach implements -D flag: power detach (force detach from elsewhere)
func handlePowerDetach(sessionName string, cmdArgs []string, config *Config) {
	sessions := session.List()

	if len(sessions) == 0 {
		// No sessions exist, create a new one if command provided
		if len(cmdArgs) > 0 {
			handleNew(sessionName, cmdArgs, config)
		} else {
			fmt.Fprintf(os.Stderr, "No screen session found.\n")
			os.Exit(1)
		}
		return
	}

	var sess *session.Session
	var err error

	if sessionName != "" {
		sess, err = session.Load(sessionName)
	if err != nil {
			// Session not found, create new one if command provided
			if len(cmdArgs) > 0 {
				handleNew(sessionName, cmdArgs, config)
			} else {
				fmt.Fprintf(os.Stderr, "No screen session found: %s\n", sessionName)
				os.Exit(1)
			}
			return
		}
	} else {
		// Find first session (prefer detached, but any will do)
		detached := findDetachedSessions(sessions)
		if len(detached) > 0 {
			sess = detached[0]
		} else {
			sess = sessions[0]
		}
	}

	// Force detach: clear PTY process reference to allow reattachment
	if sess.GetPTYProcess() != nil {
		sess.ForceDetach()
	}

	// After detaching, attach to the session
	attachToSession(sess, config)
}

// handleDetach detaches a session
func handleDetach(reattach bool, sessionName string) {
	sessions := session.List()

	if len(sessions) == 0 {
		fmt.Fprintf(os.Stderr, "No screen session found.\n")
		os.Exit(1)
	}

	var sess *session.Session
	var err error

	if sessionName != "" {
		sess, err = session.Load(sessionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No screen session found: %s\n", sessionName)
			os.Exit(1)
		}
	} else {
		// Find first attached session
		attached := findAttachedSessions(sessions)
		if len(attached) == 0 {
			fmt.Fprintf(os.Stderr, "No attached screen session found.\n")
			os.Exit(1)
		}
		sess = attached[0]
	}

	// Detach is handled by the user pressing Ctrl+A, d
	// This function just validates the session exists
	// If reattach is also requested, attach after validation
	if reattach {
		attachToSession(sess, &Config{})
	}
}

// attachToSession attaches to a session
func attachToSession(sess *session.Session, config *Config) {
	// Check if PTY process is available, try to reconnect if needed
	if sess.GetPTYProcess() == nil {
		// Try to reconnect if we have a pts path
		if sess.PtsPath != "" {
			if err := sess.ReconnectPTY(); err == nil {
				// Successfully reconnected
			} else {
				fmt.Fprintf(os.Stderr, "Error: session %s has no active PTY process\n", sess.ID)
				fmt.Fprintf(os.Stderr, "Failed to reconnect: %v\n", err)
				fmt.Fprintf(os.Stderr, "The session process may have terminated\n")
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: session %s has no active PTY process\n", sess.ID)
		fmt.Fprintf(os.Stderr, "The session may have been created in a different process\n")
		os.Exit(1)
		}
	}

	// Build attach config from main config
	attachConfig := ui.DefaultAttachConfig()
	if config != nil {
		// Parse command character
		if config.CommandChar != "" {
			cmdChar := parseCommandChar(config.CommandChar)
			if cmdChar != 0 {
				attachConfig.CommandChar = cmdChar
			}
		}
		if config.LiteralChar != "" {
			if len(config.LiteralChar) > 0 {
				attachConfig.LiteralChar = config.LiteralChar[0]
			}
		}
		attachConfig.AdaptSize = config.AdaptSize
		attachConfig.Logging = config.Logging
		attachConfig.Logfile = config.Logfile
		attachConfig.Multiuser = config.Multiuser
		attachConfig.OptimalOutput = config.OptimalOutput
		attachConfig.AllCapabilities = config.AllCapabilities
		if config.FlowControl != "" {
			attachConfig.FlowControl = config.FlowControl
		}
		attachConfig.Interrupt = config.Interrupt
		attachConfig.Term = config.Term
		attachConfig.Scrollback = config.Scrollback
	}

	err := ui.AttachWithConfig(os.Stdin, os.Stdout, os.Stderr, sess, attachConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error attaching to session: %v\n", err)
		os.Exit(1)
	}
}

// parseCommandChar parses a command character string (e.g., "^A" or "\x01")
func parseCommandChar(s string) byte {
	if len(s) == 0 {
		return 0x01 // Default: Ctrl+A
	}
	
	// Handle caret notation (^A)
	if len(s) >= 2 && s[0] == '^' {
		char := s[1]
		if char >= 'A' && char <= 'Z' {
			return char - 'A' + 1
		}
		if char >= 'a' && char <= 'z' {
			return char - 'a' + 1
		}
	}
	
	// Handle hex notation (\x01)
	if len(s) >= 4 && s[0:2] == "\\x" {
		var val byte
		fmt.Sscanf(s[2:], "%x", &val)
		return val
	}
	
	// Single character
	if len(s) == 1 {
		return s[0]
	}
	
	return 0x01 // Default
}

// handleList lists all sessions
func handleList() {
	sessions := session.List()

	if len(sessions) == 0 {
		fmt.Println("No Sockets found in /tmp/screens/S-$(whoami).")
		return
	}

	printSessionList(sessions)
}

// printSessionList prints sessions in screen-compatible format
func printSessionList(sessions []*session.Session) {
	// Screen format: "PID.TTY.HOST (Attached|Detached) DATE TIME (SESSIONNAME)"
	for _, sess := range sessions {
		status := "Detached"
		ptyProc := sess.GetPTYProcess()
		if ptyProc != nil && ptyProc.IsAlive() {
			status = "Attached"
		} else if !isProcessAliveByPID(sess.Pid) {
			status = "Dead"
		}

		// Format: PID.TTY (Status) DATE TIME (SESSIONNAME)
		tty := "pts"
		if sess.PtsPath != "" {
			parts := strings.Split(sess.PtsPath, "/")
			if len(parts) > 0 {
				tty = parts[len(parts)-1]
			}
		}

		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "localhost"
		}

		dateStr := sess.CreatedAt.Format("01/02/06")
		timeStr := sess.CreatedAt.Format("15:04:05")

		fmt.Printf("\t%d.%s.%s\t(%s)\t%s %s\t(%s)\n",
			sess.Pid, tty, hostname, status, dateStr, timeStr, sess.ID)
	}
}

// findDetachedSessions finds sessions that are not currently attached
func findDetachedSessions(sessions []*session.Session) []*session.Session {
	var detached []*session.Session
	for _, sess := range sessions {
		ptyProc := sess.GetPTYProcess()
		if ptyProc == nil || !ptyProc.IsAlive() {
			if isProcessAliveByPID(sess.Pid) {
				detached = append(detached, sess)
			}
		}
	}
	return detached
}

// findAttachedSessions finds sessions that are currently attached
func findAttachedSessions(sessions []*session.Session) []*session.Session {
	var attached []*session.Session
	for _, sess := range sessions {
		if sess.GetPTYProcess() != nil && sess.GetPTYProcess().IsAlive() {
			attached = append(attached, sess)
		}
	}
	return attached
}

// isProcessAliveByPID checks if a process is alive by PID
func isProcessAliveByPID(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// loadConfigFile loads configuration from a .screenrc file
func loadConfigFile(configFile string, config *Config) {
	// Basic config file parsing
	// For now, just check if file exists and is readable
	// Full parsing would require implementing screenrc parser
	if _, err := os.Stat(configFile); err != nil {
		if !config.Quiet {
			fmt.Fprintf(os.Stderr, "Warning: config file %s not found, using defaults\n", configFile)
		}
		return
	}
	
	// Read and parse basic config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		if !config.Quiet {
			fmt.Fprintf(os.Stderr, "Warning: could not read config file %s: %v\n", configFile, err)
		}
		return
	}
	
	// Basic parsing of common .screenrc directives
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		
		directive := parts[0]
		switch directive {
		case "escape":
			if len(parts) >= 2 {
				// Parse escape directive: escape ^Aa
				config.CommandChar = parts[1]
				if len(parts) >= 3 {
					config.LiteralChar = parts[2]
				}
			}
		case "defscrollback":
			if len(parts) >= 2 {
				if val, err := strconv.Atoi(parts[1]); err == nil {
					config.Scrollback = val
				}
			}
		case "shell":
			if len(parts) >= 2 {
				config.Shell = parts[1]
			}
		case "logfile":
			if len(parts) >= 2 {
				config.Logfile = parts[1]
				config.Logging = true
			}
		case "defflow":
			if len(parts) >= 2 {
				config.FlowControl = parts[1]
			}
		case "definterrupt":
			if len(parts) >= 2 && parts[1] == "on" {
				config.Interrupt = true
			}
		}
	}
}

func printUsage() {
	fmt.Println("sgreen - screen manager with VT100/ANSI terminal emulation")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  sgreen [options] [cmd [args]]")
	fmt.Println("    Start a new screen session with optional command")
	fmt.Println()
	fmt.Println("  sgreen -r [session]")
	fmt.Println("    Reattach to a detached session")
	fmt.Println()
	fmt.Println("  sgreen -R [session] [cmd [args]]")
	fmt.Println("    Reattach or create if none exists")
	fmt.Println()
	fmt.Println("  sgreen -RR [session] [cmd [args]]")
	fmt.Println("    Reattach or create, detaching elsewhere if needed")
	fmt.Println()
	fmt.Println("  sgreen -D [session] [cmd [args]]")
	fmt.Println("    Power detach (force detach from elsewhere)")
	fmt.Println()
	fmt.Println("  sgreen -d [session]")
	fmt.Println("    Detach a session")
	fmt.Println()
	fmt.Println("  sgreen -d -r [session]")
	fmt.Println("    Detach and reattach a session")
	fmt.Println()
	fmt.Println("  sgreen -x [session]")
	fmt.Println("    Attach to a session without detaching it (multiuser)")
	fmt.Println()
	fmt.Println("  sgreen -ls or sgreen -list")
	fmt.Println("    List all screen sessions")
	fmt.Println()
	fmt.Println("  sgreen -wipe")
	fmt.Println("    Remove dead sessions from list")
	fmt.Println()
	fmt.Println("  sgreen -v")
	fmt.Println("    Print version information")
	fmt.Println()
	fmt.Println("  sgreen -X command [session]")
	fmt.Println("    Send command to a running session")
	fmt.Println()
	fmt.Println("  sgreen -S name [cmd [args]]")
	fmt.Println("    Create a named session")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -S name        Name the session")
	fmt.Println("  -r             Reattach to a detached session")
	fmt.Println("  -R             Reattach or create if none exists")
	fmt.Println("  -RR            Reattach or create, detaching elsewhere if needed")
	fmt.Println("  -D             Power detach (force detach from elsewhere)")
	fmt.Println("  -d             Detach a session")
	fmt.Println("  -x             Attach without detaching (multiuser)")
	fmt.Println("  -s shell       Specify shell program (default: /bin/sh or $SHELL)")
	fmt.Println("  -c configfile  Use config file instead of default .screenrc")
	fmt.Println("  -e xy          Set command character (x) and literal escape (y)")
	fmt.Println("  -T term        Set TERM environment variable")
	fmt.Println("  -U             UTF-8 mode")
	fmt.Println("  -a             Include all capabilities in termcap")
	fmt.Println("  -A             Adapt window sizes to new terminal size on attach")
	fmt.Println("  -L             Turn on output logging for windows")
	fmt.Println("  -Logfile file  Log output to file")
	fmt.Println("  -H num         Set scrollback buffer size (screen uses -h)")
	fmt.Println("  -v             Print version information")
	fmt.Println("  -wipe          Remove dead sessions from list")
	fmt.Println("  -X command     Send command to a running session")
	fmt.Println("  -m             Ignore $STY environment variable")
	fmt.Println("  -O             Use optimal output mode")
	fmt.Println("  -p window      Preselect a window")
	fmt.Println("  -q             Quiet startup (suppress messages)")
	fmt.Println("  -i             Interrupt output immediately when flow control is on")
	fmt.Println("  -a             Include all capabilities in termcap")
	fmt.Println("  -f [on|off|auto] Flow control")
	fmt.Println("  -fn            Flow control off")
	fmt.Println("  -fa            Flow control automatic")
	fmt.Println("  -i             Interrupt output immediately when flow control is on")
	fmt.Println("  -O             Use optimal output mode")
	fmt.Println("  -p window      Preselect a window")
	fmt.Println("  -ls, -list     List all sessions")
	fmt.Println("  -h, -help      Show this help message")
	fmt.Println()
	fmt.Println("Inside a session, press Ctrl+A, d to detach")
}
