package ui

import (
	"testing"

	"github.com/inoki/sgreen/internal/session"
)

func TestDefaultAttachConfig(t *testing.T) {
	config := DefaultAttachConfig()
	if config == nil {
		t.Fatal("DefaultAttachConfig() returned nil")
	}

	if config.CommandChar != 0x01 {
		t.Errorf("Expected CommandChar 0x01, got %v", config.CommandChar)
	}

	if config.LiteralChar != 'a' {
		t.Errorf("Expected LiteralChar 'a', got %v", config.LiteralChar)
	}

	if config.FlowControl != "off" {
		t.Errorf("Expected FlowControl off, got %s", config.FlowControl)
	}

	if config.Scrollback != 1000 {
		t.Errorf("Expected Scrollback 1000, got %d", config.Scrollback)
	}

	if config.UTF8 != false {
		t.Errorf("Expected UTF8 false, got %v", config.UTF8)
	}

	if config.Multiuser != false {
		t.Errorf("Expected Multiuser false, got %v", config.Multiuser)
	}

	if config.AdaptSize != false {
		t.Errorf("Expected AdaptSize false, got %v", config.AdaptSize)
	}

	if config.Logging != false {
		t.Errorf("Expected Logging false, got %v", config.Logging)
	}

	if config.OptimalOutput != false {
		t.Errorf("Expected OptimalOutput false, got %v", config.OptimalOutput)
	}

	if config.AllCapabilities != false {
		t.Errorf("Expected AllCapabilities false, got %v", config.AllCapabilities)
	}

	if config.Interrupt != false {
		t.Errorf("Expected Interrupt false, got %v", config.Interrupt)
	}
}

func TestAttachConfigDefaults(t *testing.T) {
	tests := []struct {
		name    string
		checker func(*AttachConfig) bool
	}{
		{
			name: "CommandChar is Ctrl+A",
			checker: func(c *AttachConfig) bool {
				return c.CommandChar == 0x01
			},
		},
		{
			name: "LiteralChar is 'a'",
			checker: func(c *AttachConfig) bool {
				return c.LiteralChar == 'a'
			},
		},
		{
			name: "FlowControl is off",
			checker: func(c *AttachConfig) bool {
				return c.FlowControl == "off"
			},
		},
		{
			name: "Scrollback is 1000",
			checker: func(c *AttachConfig) bool {
				return c.Scrollback == 1000
			},
		},
		{
			name: "OnDetach is nil by default",
			checker: func(c *AttachConfig) bool {
				return c.OnDetach == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultAttachConfig()
			if !tt.checker(config) {
				t.Errorf("Check failed for %s", tt.name)
			}
		})
	}
}

func TestAttachConfigMutability(t *testing.T) {
	config := DefaultAttachConfig()
	if config == nil {
		t.Fatal("DefaultAttachConfig() returned nil")
	}

	originalCommandChar := config.CommandChar
	config.CommandChar = 0x02
	if config.CommandChar != 0x02 {
		t.Errorf("Failed to modify CommandChar")
	}

	config.CommandChar = originalCommandChar
	if config.CommandChar != originalCommandChar {
		t.Errorf("Failed to restore CommandChar")
	}
}

func TestAttachConfigFields(t *testing.T) {
	config := &AttachConfig{
		CommandChar:     0x02,
		LiteralChar:     'b',
		AdaptSize:       true,
		Logging:         true,
		Multiuser:       true,
		OptimalOutput:   true,
		AllCapabilities: true,
		FlowControl:     "on",
		Interrupt:       true,
		UTF8:            true,
		Encoding:        "ISO-8859-1",
		Scrollback:      5000,
		StatusLine:      true,
		StartupMessage:  true,
		Bell:            true,
		VBell:           true,
		ActivityMsg:     "Activity in %n (%t)",
		SilenceMsg:      "Silence in %n (%t)",
		SilenceTimeout:  30,
		Bindings:        map[string]string{"^A": "command"},
		ShellTitle:      "%h: %n",
		OnDetach:        func(*session.Session) {},
	}

	if config.CommandChar != 0x02 {
		t.Errorf("Expected CommandChar 0x02, got %v", config.CommandChar)
	}

	if config.LiteralChar != 'b' {
		t.Errorf("Expected LiteralChar 'b', got %v", config.LiteralChar)
	}

	if config.Scrollback != 5000 {
		t.Errorf("Expected Scrollback 5000, got %d", config.Scrollback)
	}

	if config.Bindings == nil {
		t.Errorf("Expected Bindings to be non-nil")
	}

	if config.OnDetach == nil {
		t.Errorf("Expected OnDetach to be non-nil")
	}
}
