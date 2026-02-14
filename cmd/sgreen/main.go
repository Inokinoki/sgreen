package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/inoki/sgreen/internal/session"
	"github.com/inoki/sgreen/internal/ui"
	xterm "golang.org/x/term"
)

// version is injected at build time via -ldflags "-X main.version=<version>".
// Defaults to "dev" for local builds.
var version = "dev"

// Config holds configuration options from command-line flags
type Config struct {
	Shell           string
	Term            string
	UTF8            bool
	Encoding        string
	AllCapabilities bool
	AdaptSize       bool
	Quiet           bool
	Logging         bool
	Logfile         string
	Scrollback      int
	CommandChar     string
	LiteralChar     string
	ConfigFile      string
	IgnoreSTY       bool
	OptimalOutput   bool
	PreselectWindow string
	WindowTitle     string
	LoginMode       string
	Wipe            bool
	Version         bool
	SendCommand     string
	Multiuser       bool
	FlowControl     string // "on", "off", "auto"
	Interrupt       bool
	StartupMessage  bool
	Bell            bool
	VBell           bool
	ActivityMsg     string
	SilenceMsg      string
	SilenceTimeout  int
	Bindings        map[string]string // Key bindings from config file
	Hardstatus      string            // Hardstatus line configuration
	Caption         string            // Caption line configuration
	ShellTitle      string            // Shell title format
}

func main() {
	if runDetachKeeperIfRequested() {
		return
	}

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	// Parse flags
	var (
		reattach           = flag.Bool("r", false, "Reattach to a detached session")
		reattachOrCreate   = flag.Bool("R", false, "Reattach or create if none exists")
		reattachOrCreateRR = flag.Bool("RR", false, "Reattach or create, detaching elsewhere if needed")
		powerDetach        = flag.Bool("D", false, "Power detach (force detach from elsewhere)")
		detach             = flag.Bool("d", false, "Detach a session")
		list               = flag.Bool("ls", false, "List all sessions")
		listAlt            = flag.Bool("list", false, "List all sessions (alternative)")
		sessionName        = flag.String("S", "", "Name the session")
		helpLong           = flag.Bool("help", false, "Show help")
		helpAlt            = flag.Bool("?", false, "Show help")

		// Session Configuration
		shell           = flag.String("s", "", "Shell program (default: /bin/sh or $SHELL)")
		configFile      = flag.String("c", "", "Config file instead of default .screenrc")
		escapeChars     = flag.String("e", "", "Command character and literal escape (default: ^Aa)")
		term            = flag.String("T", "", "Set TERM environment variable")
		utf8            = flag.Bool("U", false, "UTF-8 mode")
		allCapabilities = flag.Bool("a", false, "Include all capabilities in termcap")
		adaptSize       = flag.Bool("A", false, "Adapt window sizes to new terminal size on attach")

		// Output and Logging
		logging    = flag.Bool("L", false, "Turn on output logging for windows")
		logfile    = flag.String("Logfile", "", "Log output to file")
		scrollback = flag.Int("h", 0, "Set scrollback buffer size")

		// Other Options
		version         = flag.Bool("v", false, "Print version information")
		wipe            = flag.Bool("wipe", false, "Remove dead sessions from list")
		sendCommand     = flag.String("X", "", "Send command to a running session")
		ignoreSTY       = flag.Bool("m", false, "Ignore $STY environment variable")
		optimalOutput   = flag.Bool("O", false, "Use optimal output mode")
		preselectWindow = flag.String("p", "", "Preselect a window")
		windowTitle     = flag.String("t", "", "Set title for default window")
		quiet           = flag.Bool("q", false, "Quiet startup (suppress messages)")
		interrupt       = flag.Bool("i", false, "Interrupt output immediately when flow control is on")
		flowControl     = flag.String("f", "", "Flow control: on, off, or auto")
		flowControlOff  = flag.Bool("fn", false, "Flow control off")
		flowControlAuto = flag.Bool("fa", false, "Flow control automatic")
		loginOn         = flag.Bool("l", false, "Turn login mode on")
		loginOff        = flag.Bool("ln", false, "Turn login mode off")
		multiuser       = flag.Bool("x", false, "Attach to a session without detaching it (multiuser)")
	)

	flag.Usage = printUsage
	if err := flag.CommandLine.Parse(normalizeArgs(os.Args[1:])); err != nil {
		printUsage()
		os.Exit(1)
	}

	// Build config from flags
	config := &Config{
		Shell:           *shell,
		Term:            *term,
		UTF8:            *utf8,
		Encoding:        detectEncodingFromLocale(*utf8),
		AllCapabilities: *allCapabilities,
		AdaptSize:       *adaptSize,
		Quiet:           *quiet,
		Logging:         *logging,
		Logfile:         *logfile,
		Scrollback:      *scrollback,
		CommandChar:     "",
		LiteralChar:     "",
		ConfigFile:      *configFile,
		IgnoreSTY:       *ignoreSTY,
		OptimalOutput:   *optimalOutput,
		PreselectWindow: *preselectWindow,
		WindowTitle:     *windowTitle,
		Wipe:            *wipe,
		Version:         *version,
		SendCommand:     *sendCommand,
		Multiuser:       *multiuser,
		FlowControl:     *flowControl,
		Interrupt:       *interrupt,
		Bindings:        make(map[string]string),
	}

	if *loginOn {
		config.LoginMode = "on"
	}
	if *loginOff {
		config.LoginMode = "off"
	}

	// Handle -fn and -fa flags (screen-compatible)
	// These take precedence over -f value
	if *flowControlOff {
		config.FlowControl = "off"
	} else if *flowControlAuto {
		config.FlowControl = "auto"
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
		os.Exit(1)
	}

	// Handle help
	if *helpLong || *helpAlt {
		printUsage()
		os.Exit(1)
	}

	// Handle wipe
	if *wipe {
		os.Exit(handleWipe(config.Quiet))
	}

	// Handle send command (-X)
	if *sendCommand != "" {
		handleSendCommand(*sessionName, *sendCommand)
		return
	}

	// Handle list
	if *list || *listAlt {
		os.Exit(handleList(config.Quiet))
	}

	// GNU screen requires setuid-root for the owner/session form.
	if requiresSuidRootForOwnerSession(*reattach, *reattachOrCreate, *reattachOrCreateRR, *multiuser, *sessionName, flag.Args()) {
		_, _ = fmt.Fprintln(os.Stderr, "Must run suid root for multiuser support.")
		os.Exit(1)
	}

	// GNU screen requires a controlling terminal for reattach-style operations.
	if requiresTerminalForOperation(*reattach, *reattachOrCreate, *reattachOrCreateRR, *multiuser, *detach) &&
		!xterm.IsTerminal(int(os.Stdin.Fd())) {
		_, _ = fmt.Fprintln(os.Stderr, "Must be connected to a terminal.")
		os.Exit(1)
	}

	// Screen-compatible detached creation mode: -d -m (including -dmS form).
	detachedCreate := *detach && *ignoreSTY && !*reattach && !*reattachOrCreate && !*reattachOrCreateRR && !*powerDetach
	if detachedCreate {
		handleNewDetached(*sessionName, flag.Args(), config)
		return
	}

	// Screen-compatible detached no-fork creation mode: -D -m.
	detachedCreateNoFork := *powerDetach && *ignoreSTY && !*detach && !*reattach && !*reattachOrCreate && !*reattachOrCreateRR
	if detachedCreateNoFork {
		handleNewDetachedNoFork(*sessionName, flag.Args(), config)
		return
	}

	// Handle power detach (-D)
	if *powerDetach {
		targetSession := resolvePowerDetachTarget(*sessionName, flag.Args())
		handlePowerDetach(targetSession, config)
		return
	}

	// Handle detach
	if *detach {
		handleDetach(*reattach, resolveSessionName(*sessionName, flag.Args()))
		return
	}

	// Handle reattach or create with RR (-RR)
	if *reattachOrCreateRR {
		targetSession, cmdArgs := resolveSessionAndCommandArgs(*sessionName, flag.Args())
		handleReattachOrCreateRR(targetSession, cmdArgs, config)
		return
	}

	// Handle reattach or create (-R)
	if *reattachOrCreate {
		targetSession, cmdArgs := resolveSessionAndCommandArgs(*sessionName, flag.Args())
		handleReattachOrCreate(targetSession, cmdArgs, config)
		return
	}

	// Handle reattach
	if *reattach {
		targetSession := resolveSessionName(*sessionName, flag.Args())
		// Check for multiuser mode (-x)
		if *multiuser {
			// Multiuser attach: don't detach from elsewhere
			config.Multiuser = true
		}
		handleReattachWithConfig(targetSession, config)
		return
	}

	// Handle multiuser attach (-x)
	if *multiuser {
		config.Multiuser = true
		handleReattachWithConfig(resolveSessionName(*sessionName, flag.Args()), config)
		return
	}

	// Default: create new session
	handleNew(*sessionName, flag.Args(), config)
}

// printVersion prints version information
func printVersion() {
	fmt.Printf("Screen version %s (sgreen)\n", version)
}

// handleWipe removes dead sessions from the list.
// Return values mirror GNU screen CLI behavior:
// 0 when dead sessions were removed, non-zero otherwise.
func handleWipe(quiet bool) int {
	// First, clean up orphaned processes
	if err := session.CleanupOrphanedProcesses(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to cleanup orphaned processes: %v\n", err)
	}

	sessions := session.List()
	if len(sessions) == 0 {
		if !quiet {
			fmt.Printf("No Sockets found in %s.\n", screenSocketDirForDisplay())
			return 1
		}
		return 8
	}
	removed := 0

	for _, sess := range sessions {
		// Check if session is dead
		isDead := false

		// Check all windows in the session
		if len(sess.Windows) > 0 {
			allWindowsDead := true
			for _, win := range sess.Windows {
				if win.GetPTYProcess() != nil && win.GetPTYProcess().IsAlive() {
					allWindowsDead = false
					break
				}
				// Try to reconnect if we have pts path
				if win.PtsPath != "" {
					if err := sess.ReconnectPTY(); err == nil {
						allWindowsDead = false
						break
					}
				}
			}
			if allWindowsDead {
				isDead = true
			}
		} else {
			// Fallback to old method for backward compatibility
			if !isProcessAliveByPID(sess.Pid) {
				// Try to reconnect first
				if sess.PtsPath != "" {
					if err := sess.ReconnectPTY(); err != nil {
						isDead = true
					}
				} else {
					isDead = true
				}
			}
		}

		if isDead {
			// Session is dead, remove it
			if err := session.Delete(sess.ID); err == nil {
				removed++
			}
		}
	}

	if removed > 0 {
		fmt.Printf("Removed %d dead session(s)\n", removed)
		return 0
	}
	if !quiet {
		fmt.Println("No dead sessions found")
	}
	return 1
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
			_, _ = fmt.Fprintf(os.Stderr, "No screen session found.\n")
			os.Exit(1)
		}
		sess = sessions[0]
	}

	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "No screen session found.")
		os.Exit(1)
	}

	// Execute command in session
	if err := session.ExecuteCommand(sess, command); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

// handleNew creates a new session
func handleNew(sessionName string, cmdArgs []string, config *Config) {
	// Generate session name if not provided
	if sessionName == "" {
		sessionName = defaultSessionName()
	}
	requestedName := sessionName

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
	if len(cmdArgs) == 0 {
		args = ensureInteractiveShellArgs(cmdPath, args)
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
						_, _ = fmt.Fprintf(os.Stderr, "Attaching to session from $STY: %s\n", sess.ID)
					}
					attachToSession(sess, config)
					return
				}
			}
		}
	}

	// Load config file (from -c flag, $SCREENRC, or $HOME/.screenrc)
	// Only load if not explicitly disabled
	configFile := config.ConfigFile
	if configFile == "" && !config.IgnoreSTY {
		// Try to find default config file
		if found, err := findDefaultConfigFile(); err == nil && found != "" {
			configFile = found
		}
	}
	if configFile != "" {
		loadConfigFile(configFile, config)
	}
	if config != nil {
		if config.UTF8 {
			config.Encoding = "UTF-8"
		} else if config.Encoding == "" {
			config.Encoding = detectEncodingFromLocale(false)
		}
	}

	// Handle window preselection (-p)
	// Note: This is a placeholder for when multiple windows are implemented
	if config.PreselectWindow != "" {
		if !config.Quiet {
			_, _ = fmt.Fprintf(os.Stderr, "Note: Window preselection (-p) requires multiple windows feature (not yet implemented)\n")
		}
	}

	// Check if session already exists
	existingSess, err := session.Load(sessionName)
	needsPidRename := false
	if err == nil && existingSess != nil {
		if sessionHasAttachablePTY(existingSess) {
			// Session exists, try to attach to it instead
			if !config.Quiet {
				_, _ = fmt.Fprintf(os.Stderr, "Session %s already exists. Attaching...\n", sessionName)
			}
			attachToSession(existingSess, config)
			return
		}

		// Session exists but has no usable PTY; create a new unique session name.
		newName := nextAvailableSessionName(sessionName)
		if !config.Quiet {
			_, _ = fmt.Fprintf(os.Stderr, "Session %s has no active PTY. Creating new session with PID prefix.\n", sessionName)
		}
		sessionName = newName
		needsPidRename = true
	}

	// Create new session with config
	sessConfig := &session.Config{
		Term:            config.Term,
		UTF8:            config.UTF8,
		Encoding:        config.Encoding,
		Scrollback:      config.Scrollback,
		AllCapabilities: config.AllCapabilities,
	}
	sess, err := session.NewWithConfig(sessionName, cmdPath, args, sessConfig)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}
	if needsPidRename {
		pidName := fmt.Sprintf("%d-%s", sess.Pid, requestedName)
		if pidName != sessionName {
			if err := sess.Rename(pidName); err != nil {
				if !config.Quiet {
					_, _ = fmt.Fprintf(os.Stderr, "Warning: created session %s, but failed to rename to %s: %v\n", sessionName, pidName, err)
				}
			} else if !config.Quiet {
				_, _ = fmt.Fprintf(os.Stderr, "Session created as %s.\n", pidName)
			}
		}
	}

	applyWindowTitle(sess, config)

	// Attach to the new session
	attachToSession(sess, config)
}

// handleNewDetached creates a new session without attaching (screen -d -m/-dmS behavior).
func handleNewDetached(sessionName string, cmdArgs []string, config *Config) {
	if config == nil {
		config = &Config{}
	}

	if sessionName == "" {
		sessionName = defaultSessionName()
	}

	// Determine shell
	shellPath := "/bin/sh"
	if config.Shell != "" {
		shellPath = config.Shell
	} else if envShell := os.Getenv("SHELL"); envShell != "" {
		shellPath = envShell
	}

	cmdPath := shellPath
	args := cmdArgs
	if len(cmdArgs) > 0 {
		cmdPath = cmdArgs[0]
		args = cmdArgs[1:]
	}
	if len(cmdArgs) == 0 {
		args = ensureInteractiveShellArgs(cmdPath, args)
	}

	// Load config file when available.
	configFile := config.ConfigFile
	if configFile == "" {
		if found, err := findDefaultConfigFile(); err == nil && found != "" {
			configFile = found
		}
	}
	if configFile != "" {
		loadConfigFile(configFile, config)
	}
	if config.UTF8 {
		config.Encoding = "UTF-8"
	} else if config.Encoding == "" {
		config.Encoding = detectEncodingFromLocale(false)
	}

	sessConfig := &session.Config{
		Term:            config.Term,
		UTF8:            config.UTF8,
		Encoding:        config.Encoding,
		Scrollback:      config.Scrollback,
		AllCapabilities: config.AllCapabilities,
	}
	sess, err := session.NewWithConfig(sessionName, cmdPath, args, sessConfig)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}
	applyWindowTitle(sess, config)

	// Keep PTY master alive after this process exits (same mechanism as detach).
	startDetachKeeper(sess)
	sess.ForceDetach()
}

// handleNewDetachedNoFork creates a detached session without spawning a keeper process.
// This mirrors GNU screen -D -m behavior by waiting until the session command exits.
func handleNewDetachedNoFork(sessionName string, cmdArgs []string, config *Config) {
	if config == nil {
		config = &Config{}
	}

	if sessionName == "" {
		sessionName = defaultSessionName()
	}

	// Determine shell
	shellPath := "/bin/sh"
	if config.Shell != "" {
		shellPath = config.Shell
	} else if envShell := os.Getenv("SHELL"); envShell != "" {
		shellPath = envShell
	}

	cmdPath := shellPath
	args := cmdArgs
	if len(cmdArgs) > 0 {
		cmdPath = cmdArgs[0]
		args = cmdArgs[1:]
	}
	if len(cmdArgs) == 0 {
		args = ensureInteractiveShellArgs(cmdPath, args)
	}

	// Load config file when available.
	configFile := config.ConfigFile
	if configFile == "" {
		if found, err := findDefaultConfigFile(); err == nil && found != "" {
			configFile = found
		}
	}
	if configFile != "" {
		loadConfigFile(configFile, config)
	}
	if config.UTF8 {
		config.Encoding = "UTF-8"
	} else if config.Encoding == "" {
		config.Encoding = detectEncodingFromLocale(false)
	}

	sessConfig := &session.Config{
		Term:            config.Term,
		UTF8:            config.UTF8,
		Encoding:        config.Encoding,
		Scrollback:      config.Scrollback,
		AllCapabilities: config.AllCapabilities,
	}
	sess, err := session.NewWithConfig(sessionName, cmdPath, args, sessConfig)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		os.Exit(1)
	}
	applyWindowTitle(sess, config)

	ptyProc := sess.GetPTYProcess()
	sess.ForceDetach()
	if ptyProc != nil {
		_ = ptyProc.Wait()
	}
}

func applyWindowTitle(sess *session.Session, config *Config) {
	if sess == nil || config == nil || config.WindowTitle == "" {
		return
	}
	win := sess.GetCurrentWindow()
	if win == nil {
		return
	}
	win.Title = config.WindowTitle
	_ = sess.Save()
}

func sessionHasAttachablePTY(sess *session.Session) bool {
	if sess == nil {
		return false
	}
	if ptyProc := sess.GetPTYProcess(); ptyProc != nil && ptyProc.IsAlive() {
		return true
	}
	if sess.PtsPath != "" {
		if err := sess.ReconnectPTY(); err == nil {
			return true
		}
	}
	return false
}

func nextAvailableSessionName(base string) string {
	for i := 0; ; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		if sess, err := session.Load(candidate); err != nil || sess == nil {
			return candidate
		}
	}
}

// handleReattachWithConfig reattaches with configuration
func handleReattachWithConfig(sessionName string, config *Config) {
	sessions := session.List()
	if config == nil {
		config = &Config{}
	}

	sess, errMsg, printList := selectReattachSession(
		sessions,
		sessionName,
		config.Multiuser,
		session.Load,
		isSessionAttached,
	)
	if errMsg != "" {
		if config.Quiet && !config.Multiuser {
			if isNoResumableError(errMsg) {
				os.Exit(10)
			}
			if printList && strings.Contains(errMsg, "There are several detached sessions:") {
				count := resumableSessionCount(sessions)
				if count < 2 {
					count = 2
				}
				os.Exit(10 + count)
			}
		}
		_, _ = fmt.Fprintln(os.Stderr, errMsg)
		if printList {
			printSessionList(sessions)
		}
		os.Exit(1)
	}

	attachToSession(sess, config)
}

func isSessionAttached(sess *session.Session) bool {
	if sess == nil {
		return false
	}
	ptyProc := sess.GetPTYProcess()
	return ptyProc != nil && ptyProc.IsAlive()
}

func selectReattachSession(
	sessions []*session.Session,
	sessionName string,
	multiuser bool,
	loadByName func(string) (*session.Session, error),
	isAttached func(*session.Session) bool,
) (*session.Session, string, bool) {
	if len(sessions) == 0 {
		if multiuser {
			return nil, noAttachableScreenMessage(sessionName), false
		}
		return nil, noResumableScreenMessage(sessionName), false
	}

	if sessionName != "" {
		sess, err := loadByName(sessionName)
		if err != nil {
			if multiuser {
				return nil, noAttachableScreenMessage(sessionName), false
			}
			return nil, noResumableScreenMessage(sessionName), false
		}
		if !multiuser && isAttached(sess) {
			return nil, fmt.Sprintf("Session %s is attached; use -d -r or -x.", sessionName), false
		}
		return sess, "", false
	}

	if multiuser {
		if len(sessions) == 1 {
			return sessions[0], "", false
		}
		return nil, "Multiple sessions found. Specify session name with -x:", true
	}

	detached := make([]*session.Session, 0, len(sessions))
	for _, sess := range sessions {
		if !isAttached(sess) && sessionHasAliveProcess(sess) {
			detached = append(detached, sess)
		}
	}
	if len(detached) == 1 {
		return detached[0], "", false
	}
	if len(detached) > 1 {
		return nil, "There are several detached sessions:", true
	}
	if len(sessions) == 1 && isAttached(sessions[0]) {
		return nil, noResumableScreenMessage(""), false
	}
	return nil, noResumableScreenMessage(""), false
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
func handlePowerDetach(sessionName string, config *Config) {
	sessions := session.List()
	_ = config

	if len(sessions) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, noDetachableScreenMessage(sessionName))
		os.Exit(1)
		return
	}

	var sess *session.Session
	var err error

	if sessionName != "" {
		sess, err = session.Load(sessionName)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, noDetachableScreenMessage(sessionName))
			os.Exit(1)
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
		_, _ = fmt.Fprintln(os.Stderr, noDetachableScreenMessage(sessionName))
		os.Exit(1)
	}

	var sess *session.Session
	var err error

	if sessionName != "" {
		sess, err = session.Load(sessionName)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, noDetachableScreenMessage(sessionName))
			os.Exit(1)
		}
	} else {
		// Find first attached session
		attached := findAttachedSessions(sessions)
		if len(attached) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, noDetachableScreenMessage(""))
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
	// Permission check for multi-user sessions
	if sess.Owner != "" || len(sess.AllowedUsers) > 0 {
		user := session.CurrentUser()
		if !sess.CanAttach(user) {
			_, _ = fmt.Fprintf(os.Stderr, "Permission denied: user %s is not allowed to attach to session %s\n", user, sess.ID)
			os.Exit(1)
		}
	}

	// Check if PTY process is available, try to reconnect if needed
	if sess.GetPTYProcess() == nil {
		// Try to reconnect if we have a pts path
		if sess.PtsPath != "" {
			if err := sess.ReconnectPTY(); err == nil {
				// Successfully reconnected
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "Error: session %s has no active PTY process\n", sess.ID)
				_, _ = fmt.Fprintf(os.Stderr, "Failed to reconnect: %v\n", err)
				_, _ = fmt.Fprintf(os.Stderr, "The session process may have terminated\n")
				os.Exit(1)
			}
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "Error: session %s has no active PTY process\n", sess.ID)
			_, _ = fmt.Fprintf(os.Stderr, "The session may have been created in a different process\n")
			os.Exit(1)
		}
	}

	// Build attach config from main config
	attachConfig := ui.DefaultAttachConfig()
	startedKeeper := false
	onDetach := func(detachSess *session.Session) {
		if startedKeeper {
			return
		}
		startedKeeper = true
		startDetachKeeper(detachSess)
	}
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
		attachConfig.UTF8 = config.UTF8
		attachConfig.Encoding = config.Encoding
		attachConfig.Scrollback = config.Scrollback
		// Enable status line if hardstatus or caption is configured
		if config.Hardstatus != "" {
			attachConfig.StatusLine = true
			attachConfig.StatusFormat = config.Hardstatus
		} else if config.Caption != "" {
			attachConfig.StatusLine = true
			attachConfig.StatusFormat = config.Caption
		} else {
			attachConfig.StatusLine = false
			attachConfig.StatusFormat = ""
		}
		// Startup message and bell settings
		attachConfig.StartupMessage = config.StartupMessage
		attachConfig.Bell = config.Bell
		attachConfig.VBell = config.VBell
		// Activity and silence monitoring
		attachConfig.ActivityMsg = config.ActivityMsg
		attachConfig.SilenceMsg = config.SilenceMsg
		attachConfig.SilenceTimeout = config.SilenceTimeout
		// Key bindings
		if config.Bindings != nil {
			attachConfig.Bindings = make(map[string]string)
			for k, v := range config.Bindings {
				attachConfig.Bindings[k] = v
			}
		}
		// Shell title format
		attachConfig.ShellTitle = config.ShellTitle
	}
	attachConfig.OnDetach = onDetach

	err := ui.AttachWithConfig(os.Stdin, os.Stdout, os.Stderr, sess, attachConfig)
	if err == nil || err == ui.ErrDetach {
		onDetach(sess)
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "Error attaching to session: %v\n", err)
	os.Exit(1)
}

func runDetachKeeperIfRequested() bool {
	if os.Getenv("SGREEN_DETACH_KEEPER") != "1" {
		return false
	}
	debugDetachKeeper("keeper: starting")
	fdStr := os.Getenv("SGREEN_HOLD_FD")
	fd, err := strconv.Atoi(fdStr)
	if err != nil || fd <= 0 {
		debugDetachKeeper("keeper: invalid SGREEN_HOLD_FD=%q", fdStr)
		return true
	}
	readyStr := os.Getenv("SGREEN_READY_FD")
	readyFD, readyErr := strconv.Atoi(readyStr)
	if readyErr == nil && readyFD > 0 {
		if readyFile := os.NewFile(uintptr(readyFD), "sgreen-keeper-ready"); readyFile != nil {
			_, _ = readyFile.Write([]byte("ready\n"))
			_ = readyFile.Close()
		}
	}
	file := os.NewFile(uintptr(fd), "sgreen-pty-master")
	if file == nil {
		debugDetachKeeper("keeper: failed to open fd=%d", fd)
		return true
	}
	debugDetachKeeper("keeper: holding fd=%d", fd)
	// Keep the PTY master open so detached processes do not receive SIGHUP.
	select {}
}

func startDetachKeeper(sess *session.Session) {
	if sess == nil {
		return
	}
	ptyProc := sess.GetPTYProcess()
	if ptyProc == nil {
		if win := sess.GetCurrentWindow(); win != nil {
			ptyProc = win.GetPTYProcess()
		}
	}
	if ptyProc == nil || ptyProc.Pty == nil {
		debugDetachKeeper("keeper: no PTY to hold for session %q", sess.ID)
		return
	}
	selfPath, err := os.Executable()
	if err != nil {
		debugDetachKeeper("keeper: failed to get executable path: %v", err)
		return
	}
	cmd := exec.Command(selfPath)
	readyR, readyW, err := os.Pipe()
	if err != nil {
		debugDetachKeeper("keeper: failed to create ready pipe: %v", err)
		return
	}
	defer readyR.Close()
	cmd.Env = append(os.Environ(),
		"SGREEN_DETACH_KEEPER=1",
		"SGREEN_HOLD_FD=3",
		"SGREEN_READY_FD=4",
	)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.ExtraFiles = []*os.File{ptyProc.Pty, readyW}
	setDetachSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		debugDetachKeeper("keeper: failed to start: %v", err)
		_ = readyW.Close()
		return
	}
	_ = readyW.Close()
	waitForKeeperReady(readyR)
	debugDetachKeeper("keeper: started pid=%d for session %q", cmd.Process.Pid, sess.ID)
}

func debugDetachKeeper(format string, args ...any) {
	if os.Getenv("SGREEN_KEEPER_DEBUG") == "" {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func waitForKeeperReady(readyR *os.File) {
	if readyR == nil {
		return
	}
	buf := make([]byte, 16)
	_ = readyR.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = readyR.Read(buf)
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
		if _, err := fmt.Sscanf(s[2:], "%x", &val); err == nil {
			return val
		}
		return 0x01
	}

	// Single character
	if len(s) == 1 {
		return s[0]
	}

	return 0x01 // Default
}

// detectEncodingFromLocale detects encoding from locale environment variables.
func detectEncodingFromLocale(forceUTF8 bool) string {
	if forceUTF8 {
		return "UTF-8"
	}
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		locale := os.Getenv(key)
		if locale == "" {
			continue
		}
		parts := strings.Split(locale, ".")
		if len(parts) < 2 {
			continue
		}
		encoding := strings.ToUpper(parts[1])
		encoding = strings.ReplaceAll(encoding, "_", "-")
		switch encoding {
		case "UTF-8", "UTF8":
			return "UTF-8"
		case "ISO-8859-1", "ISO8859-1", "LATIN1":
			return "ISO-8859-1"
		case "ISO-8859-2", "ISO8859-2", "LATIN2":
			return "ISO-8859-2"
		case "ISO-8859-15", "ISO8859-15", "LATIN9":
			return "ISO-8859-15"
		case "WINDOWS-1252", "CP1252":
			return "WINDOWS-1252"
		case "WINDOWS-1251", "CP1251":
			return "WINDOWS-1251"
		case "KOI8-R", "KOI8R":
			return "KOI8-R"
		case "KOI8-U", "KOI8U":
			return "KOI8-U"
		}
	}
	return "UTF-8"
}

// handleList lists all sessions.
// Return codes follow GNU screen conventions as closely as practical:
// 0 when sessions are listed, 1 when none are found, 8 for quiet no-session listing.
func handleList(quiet bool) int {
	allSessions := session.List()
	sessions := listableSessions(allSessions)

	if len(sessions) == 0 {
		if quiet {
			return 8
		}
		fmt.Printf("No Sockets found in %s.\n", screenSocketDirForDisplay())
		return 1
	}

	if quiet {
		return 0
	}

	entries := sessionListEntries(sessions)
	if len(entries) == 0 {
		fmt.Printf("No Sockets found in %s.\n", screenSocketDirForDisplay())
		return 1
	}

	if len(entries) == 1 {
		fmt.Println("There is a screen on:")
	} else {
		fmt.Println("There are screens on:")
	}
	for _, entry := range entries {
		fmt.Println(entry)
	}
	fmt.Printf("%d %s in %s.\n", len(entries), socketWord(len(entries)), screenSocketDirForDisplay())
	return 0
}

func listableSessions(sessions []*session.Session) []*session.Session {
	listable := make([]*session.Session, 0, len(sessions))
	for _, sess := range sessions {
		if sess == nil {
			continue
		}
		if isSessionAttached(sess) || sessionHasAliveProcess(sess) {
			listable = append(listable, sess)
		}
	}
	return listable
}

// printSessionList prints sessions in screen-compatible format
func printSessionList(sessions []*session.Session) {
	for _, entry := range sessionListEntries(sessions) {
		fmt.Println(entry)
	}
}

func sessionListEntries(sessions []*session.Session) []string {
	// Screen format: "PID.TTY.HOST (Attached|Detached) DATE TIME (SESSIONNAME)"
	nameCounts := make(map[string]int, len(sessions))
	for _, sess := range sessions {
		nameCounts[sess.ID]++
	}

	entries := make([]string, 0, len(sessions))
	for _, sess := range sessions {
		status := "Detached"
		ptyProc := sess.GetPTYProcess()
		if ptyProc != nil && ptyProc.IsAlive() {
			status = "Attached"
		} else if !sessionHasAliveProcess(sess) {
			status = "Dead"
		}
		if status == "Dead" {
			continue
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

		displayName := sess.ID
		pidPrefix := strconv.Itoa(sess.Pid) + "-"
		baseName := sess.ID
		if strings.HasPrefix(sess.ID, pidPrefix) {
			baseName = strings.TrimPrefix(sess.ID, pidPrefix)
		}
		if strings.HasPrefix(sess.ID, pidPrefix) || nameCounts[sess.ID] > 1 {
			displayName = fmt.Sprintf("%d.%s", sess.Pid, baseName)
		}

		entries = append(entries, fmt.Sprintf("\t%d.%s.%s\t(%s)\t%s %s\t(%s)",
			sess.Pid, tty, hostname, status, dateStr, timeStr, displayName))
	}
	return entries
}

func screenSocketDirForDisplay() string {
	if screenDir := os.Getenv("SCREENDIR"); screenDir != "" {
		return screenDir
	}
	user := session.CurrentUser()
	if user == "" {
		user = "unknown"
	}
	return filepath.Join(os.TempDir(), "screens", "S-"+user)
}

func socketWord(count int) string {
	if count == 1 {
		return "Socket"
	}
	return "Sockets"
}

func resolveSessionName(flagValue string, args []string) string {
	if flagValue != "" {
		return flagValue
	}
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func resolveSessionAndCommandArgs(flagValue string, args []string) (string, []string) {
	if flagValue != "" {
		return flagValue, args
	}
	if len(args) == 0 {
		return "", nil
	}
	return args[0], args[1:]
}

func resolvePowerDetachTarget(flagValue string, args []string) string {
	if flagValue != "" {
		return flagValue
	}
	// GNU screen treats "-D <name>" as a named target, but additional
	// trailing arguments are not command args for -D and should not turn the
	// operation into a named detach.
	if len(args) == 1 {
		return args[0]
	}
	return ""
}

func requiresSuidRootForOwnerSession(reattach bool, reattachOrCreate bool, reattachOrCreateRR bool, multiuser bool, sessionFlag string, args []string) bool {
	var target string
	switch {
	case reattach || multiuser:
		target = resolveSessionName(sessionFlag, args)
	case reattachOrCreate || reattachOrCreateRR:
		target, _ = resolveSessionAndCommandArgs(sessionFlag, args)
	default:
		return false
	}
	return isOwnerSessionTarget(target)
}

func isOwnerSessionTarget(name string) bool {
	if name == "" {
		return false
	}
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	current := session.CurrentUser()
	if current == "" {
		return true
	}
	return parts[0] != current
}

func isNoResumableError(msg string) bool {
	return strings.HasPrefix(msg, "There is no screen to be resumed")
}

func resumableSessionCount(sessions []*session.Session) int {
	count := 0
	for _, sess := range sessions {
		if sess == nil {
			continue
		}
		if !isSessionAttached(sess) && sessionHasAliveProcess(sess) {
			count++
		}
	}
	return count
}

func normalizeArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		switch {
		case arg == "-dmS":
			normalized = append(normalized, "-d", "-m", "-S")
		case strings.HasPrefix(arg, "-dmS") && len(arg) > 4:
			normalized = append(normalized, "-d", "-m", "-S", arg[4:])
		case arg == "-dm":
			normalized = append(normalized, "-d", "-m")
		default:
			normalized = append(normalized, arg)
		}
	}
	return normalized
}

func requiresTerminalForOperation(reattach bool, reattachOrCreate bool, reattachOrCreateRR bool, multiuser bool, detach bool) bool {
	return reattach || reattachOrCreate || reattachOrCreateRR || multiuser || (detach && reattach)
}

func noResumableScreenMessage(name string) string {
	if name != "" {
		return fmt.Sprintf("There is no screen to be resumed matching %s.", name)
	}
	return "There is no screen to be resumed."
}

func noAttachableScreenMessage(name string) string {
	if name != "" {
		return fmt.Sprintf("There is no screen to be attached matching %s.", name)
	}
	return "There is no screen to be attached."
}

func noDetachableScreenMessage(name string) string {
	if name != "" {
		return fmt.Sprintf("There is no screen to be detached matching %s.", name)
	}
	return "There is no screen to be detached."
}

func defaultSessionName() string {
	// GNU screen default style: <pid>.<tty>.<host>.
	// If no controlling TTY is available, tty component is empty.
	tty := sanitizeSessionNameComponent(detectTTYName())
	host := sanitizeSessionNameComponent(defaultHostName())
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("%d.%s.%s", os.Getpid(), tty, host)
}

func defaultHostName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "localhost"
	}
	return host
}

func detectTTYName() string {
	link, err := os.Readlink("/dev/fd/0")
	if err != nil || link == "" {
		return ""
	}
	if !strings.HasPrefix(link, "/dev/") {
		return ""
	}
	base := filepath.Base(link)
	if base == "" || base == "0" || strings.HasPrefix(base, "fd") {
		return ""
	}
	return base
}

func sanitizeSessionNameComponent(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// findDetachedSessions finds sessions that are not currently attached
func findDetachedSessions(sessions []*session.Session) []*session.Session {
	var detached []*session.Session
	for _, sess := range sessions {
		ptyProc := sess.GetPTYProcess()
		if ptyProc == nil || !ptyProc.IsAlive() {
			if sessionHasAliveProcess(sess) {
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

func ensureInteractiveShellArgs(cmdPath string, args []string) []string {
	if len(args) > 0 {
		return args
	}
	switch strings.ToLower(filepath.Base(cmdPath)) {
	case "zsh", "bash", "sh", "ksh", "fish":
		return []string{"-i"}
	default:
		return args
	}
}

func sessionHasAliveProcess(sess *session.Session) bool {
	if sess == nil {
		return false
	}
	for _, win := range sess.Windows {
		if win == nil {
			continue
		}
		if win.Pid > 0 && isProcessAliveByPID(win.Pid) {
			return true
		}
	}
	return sess.Pid > 0 && isProcessAliveByPID(sess.Pid)
}

// findDefaultConfigFile finds the default config file location
func findDefaultConfigFile() (string, error) {
	// Check $SCREENRC environment variable
	if screenrc := os.Getenv("SCREENRC"); screenrc != "" {
		if _, err := os.Stat(screenrc); err == nil {
			return screenrc, nil
		}
	}

	// Check $HOME/.screenrc
	homeDir, err := os.UserHomeDir()
	if err == nil {
		screenrc := filepath.Join(homeDir, ".screenrc")
		if _, err := os.Stat(screenrc); err == nil {
			return screenrc, nil
		}
	}

	// Check system-wide config
	if systemScreenrc := os.Getenv("SYSTEM_SCREENRC"); systemScreenrc != "" {
		if _, err := os.Stat(systemScreenrc); err == nil {
			return systemScreenrc, nil
		}
	}

	return "", nil
}

// loadConfigFile loads configuration from a .screenrc file
func loadConfigFile(configFile string, config *Config) {
	if _, err := os.Stat(configFile); err != nil {
		if !config.Quiet {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: config file %s not found, using defaults\n", configFile)
		}
		return
	}

	// Read and parse config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		if !config.Quiet {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: could not read config file %s: %v\n", configFile, err)
		}
		return
	}

	// Parse config file with enhanced parser
	lines := strings.Split(string(data), "\n")
	processedFiles := make(map[string]bool)
	baseDir := filepath.Dir(configFile)

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle line continuation
		if strings.HasSuffix(line, "\\") {
			line = strings.TrimSuffix(line, "\\")
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				line = line + " " + nextLine
			}
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		directive := parts[0]
		args := parts[1:]

		switch directive {
		case "source", "sourcefile":
			// Handle source directive
			if len(args) > 0 {
				sourceFile := args[0]
				if !filepath.IsAbs(sourceFile) {
					sourceFile = filepath.Join(baseDir, sourceFile)
				}

				if processedFiles[sourceFile] {
					continue
				}
				processedFiles[sourceFile] = true

				// Recursively load source file
				loadConfigFile(sourceFile, config)
			}

		case "escape":
			if len(args) >= 1 {
				escapeStr := args[0]
				// Parse escape string like "^Aa"
				if len(escapeStr) >= 2 {
					config.CommandChar = escapeStr[:1]
					config.LiteralChar = escapeStr[1:2]
				}
			}

		case "shell":
			if len(args) >= 1 {
				config.Shell = strings.Join(args, " ")
			}

		case "defscrollback":
			if len(args) >= 1 {
				if val, err := strconv.Atoi(args[0]); err == nil {
					config.Scrollback = val
				}
			}

		case "logfile":
			if len(args) >= 1 {
				config.Logfile = strings.Join(args, " ")
				config.Logging = true
			}

		case "log":
			if len(args) >= 1 && args[0] == "on" {
				config.Logging = true
			} else if len(args) >= 1 && args[0] == "off" {
				config.Logging = false
			}

		case "defflow":
			if len(args) >= 1 {
				config.FlowControl = args[0]
			}

		case "definterrupt":
			if len(args) >= 1 && args[0] == "on" {
				config.Interrupt = true
			} else if len(args) >= 1 && args[0] == "off" {
				config.Interrupt = false
			}

		case "startup_message":
			if len(args) >= 1 && args[0] == "off" {
				config.StartupMessage = false
			} else {
				config.StartupMessage = true
			}

		case "bell":
			if len(args) >= 1 && args[0] == "off" {
				config.Bell = false
			} else {
				config.Bell = true
			}

		case "vbell":
			if len(args) >= 1 && args[0] == "off" {
				config.VBell = false
			} else {
				config.VBell = true
			}

		case "activity":
			if len(args) >= 1 {
				config.ActivityMsg = strings.Join(args, " ")
			} else {
				config.ActivityMsg = "Activity in window %n"
			}

		case "silence":
			if len(args) >= 1 {
				config.SilenceMsg = strings.Join(args, " ")
			} else {
				config.SilenceMsg = "Silence in window %n"
			}
			// Default silence timeout is 30 seconds if not specified
			if config.SilenceTimeout == 0 {
				config.SilenceTimeout = 30
			}

		case "hardstatus":
			// Parse hardstatus configuration
			// Format: hardstatus [on|off] or hardstatus string [format]
			if len(args) >= 1 {
				if args[0] == "on" || args[0] == "off" {
					// Toggle format - for now, just enable if "on"
					if args[0] == "on" && config.Hardstatus == "" {
						config.Hardstatus = "%h" // Default format
					} else if args[0] == "off" {
						config.Hardstatus = ""
					}
				} else if args[0] == "string" && len(args) >= 2 {
					// Format: hardstatus string <format>
					config.Hardstatus = strings.Join(args[1:], " ")
				} else {
					// Assume it's a format string
					config.Hardstatus = strings.Join(args, " ")
				}
			}

		case "caption":
			// Parse caption configuration
			// Format: caption [always|splitonly] or caption string [format]
			if len(args) >= 1 {
				if args[0] == "string" && len(args) >= 2 {
					// Format: caption string <format>
					config.Caption = strings.Join(args[1:], " ")
				} else if args[0] != "always" && args[0] != "splitonly" {
					// Assume it's a format string
					config.Caption = strings.Join(args, " ")
				}
			}

		case "shelltitle":
			// Store shelltitle format
			if len(args) >= 1 {
				config.ShellTitle = strings.Join(args, " ")
			}

		case "bind", "bindkey":
			// Store key bindings: bind key command
			if len(args) >= 2 {
				key := args[0]
				command := strings.Join(args[1:], " ")
				config.Bindings[key] = command
			}

		case "unbind", "unbindkey":
			// Remove key binding
			if len(args) >= 1 {
				delete(config.Bindings, args[0])
			}
		}
	}
}

func printUsage() {
	prog := os.Args[0]
	if prog == "" {
		prog = "sgreen"
	}
	fmt.Printf("Use: %s [-opts] [cmd [args]]\n", prog)
	fmt.Printf(" or: %s -r [host.tty]\n", prog)
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
	fmt.Println("  -t title       Set title for default window")
	fmt.Println("  -T term        Set TERM environment variable")
	fmt.Println("  -U             UTF-8 mode")
	fmt.Println("  -a             Include all capabilities in termcap")
	fmt.Println("  -A             Adapt window sizes to new terminal size on attach")
	fmt.Println("  -L             Turn on output logging for windows")
	fmt.Println("  -Logfile file  Log output to file")
	fmt.Println("  -h num         Set scrollback buffer size")
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
	fmt.Println("  -l             Turn login mode on")
	fmt.Println("  -ln            Turn login mode off")
	fmt.Println("  -i             Interrupt output immediately when flow control is on")
	fmt.Println("  -O             Use optimal output mode")
	fmt.Println("  -p window      Preselect a window")
	fmt.Println("  -ls, -list     List all sessions")
	fmt.Println("  -help, -?      Show this help message")
	fmt.Println()
	fmt.Println("Inside a session, press Ctrl+A, d to detach")
}
