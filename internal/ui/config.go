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
	Scrollback     int    // Scrollback buffer size
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
		Scrollback: 1000,
	}
}

