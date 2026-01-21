package ui

// AttachConfig holds configuration for attaching to a session
type AttachConfig struct {
	CommandChar    byte   // Command character (default: 0x01 = Ctrl+A)
	LiteralChar    byte   // Literal escape character (default: 'a')
	AdaptSize      bool   // Adapt window sizes to new terminal size
	Logging        bool   // Enable output logging
	Logfile        string // Log file path
	Multiuser      bool   // Allow multiuser attach
	OptimalOutput  bool   // Use optimal output mode
	AllCapabilities bool  // Include all capabilities in termcap
	FlowControl    string // Flow control: "on", "off", "auto"
	Interrupt      bool   // Interrupt output immediately when flow control is on
	Term           string // Terminal type (for window creation)
	UTF8           bool   // UTF-8 mode
	Encoding       string // Window encoding (e.g., UTF-8, ISO-8859-1)
	Scrollback     int    // Scrollback buffer size
	StatusLine     bool   // Enable status line
	StatusFormat   string // Status line format string
	StartupMessage bool   // Show startup message
	Bell           bool   // Enable bell
	VBell          bool   // Enable visual bell
	ActivityMsg    string            // Activity message template
	SilenceMsg     string            // Silence message template
	SilenceTimeout int               // Silence timeout in seconds
	Bindings       map[string]string // Custom key bindings (key -> command)
	ShellTitle     string            // Shell title format
}

// DefaultAttachConfig returns default attach configuration
func DefaultAttachConfig() *AttachConfig {
	return &AttachConfig{
		CommandChar: 0x01, // Ctrl+A
		LiteralChar: 'a',
		AdaptSize:   false,
		Logging:     false,
		Multiuser:   false,
		OptimalOutput: false,
		AllCapabilities: false,
		FlowControl: "off",
		Interrupt: false,
		UTF8: false,
		Encoding: "",
		Scrollback: 1000,
	}
}

