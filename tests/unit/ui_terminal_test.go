package unit

import (
	"os"
	"testing"

	"github.com/inoki/sgreen/internal/ui"
)

func TestDetectTerminalCapabilitiesDefault(t *testing.T) {
	os.Unsetenv("TERM")
	os.Unsetenv("COLORTERM")

	caps := ui.DetectTerminalCapabilities()

	if caps.HasColor || caps.Supports256Color || caps.SupportsTrueColor {
		t.Errorf("Default terminal should have no capabilities set")
	}
}

func TestDetectTerminalCapabilitiesColor(t *testing.T) {
	os.Setenv("TERM", "xterm-color")
	defer os.Unsetenv("TERM")

	caps := ui.DetectTerminalCapabilities()

	if !caps.HasColor {
		t.Errorf("Expected HasColor to be true for xterm-color")
	}
}

func TestDetectTerminalCapabilities256Color(t *testing.T) {
	os.Setenv("TERM", "xterm-256color")
	defer os.Unsetenv("TERM")

	caps := ui.DetectTerminalCapabilities()

	if !caps.Supports256Color {
		t.Errorf("Expected Supports256Color to be true for xterm-256color")
	}
}

func TestDetectTerminalCapabilitiesXterm(t *testing.T) {
	os.Setenv("TERM", "xterm")
	defer os.Unsetenv("TERM")

	caps := ui.DetectTerminalCapabilities()

	if !caps.SupportsMouse {
		t.Errorf("Expected SupportsMouse to be true for xterm")
	}
	if !caps.SupportsBracketedPaste {
		t.Errorf("Expected SupportsBracketedPaste to be true for xterm")
	}
	if !caps.SupportsAltScreen {
		t.Errorf("Expected SupportsAltScreen to be true for xterm")
	}
}

func TestDetectTerminalCapabilitiesScreen(t *testing.T) {
	os.Setenv("TERM", "screen")
	defer os.Unsetenv("TERM")

	caps := ui.DetectTerminalCapabilities()

	if !caps.SupportsMouse {
		t.Errorf("Expected SupportsMouse to be true for screen")
	}
	if !caps.SupportsCursor {
		t.Errorf("Expected SupportsCursor to be true for screen")
	}
}

func TestDetectTerminalCapabilitiesTmux(t *testing.T) {
	os.Setenv("TERM", "tmux")
	defer os.Unsetenv("TERM")

	caps := ui.DetectTerminalCapabilities()

	if !caps.SupportsMouse {
		t.Errorf("Expected SupportsMouse to be true for tmux")
	}
	if !caps.SupportsAltScreen {
		t.Errorf("Expected SupportsAltScreen to be true for tmux")
	}
}

func TestDetectTerminalCapabilitiesTrueColor(t *testing.T) {
	os.Setenv("COLORTERM", "truecolor")
	defer os.Unsetenv("COLORTERM")

	caps := ui.DetectTerminalCapabilities()

	if !caps.SupportsTrueColor {
		t.Errorf("Expected SupportsTrueColor to be true for COLORTERM=truecolor")
	}
	if !caps.HasColor {
		t.Errorf("Expected HasColor to be true with COLORTERM set")
	}
}

func TestDetectTerminalCapabilities24bit(t *testing.T) {
	os.Setenv("COLORTERM", "24bit")
	defer os.Unsetenv("COLORTERM")

	caps := ui.DetectTerminalCapabilities()

	if !caps.SupportsTrueColor {
		t.Errorf("Expected SupportsTrueColor to be true for COLORTERM=24bit")
	}
}

func TestDetectTerminalCapabilitiesCombined(t *testing.T) {
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	defer func() {
		os.Unsetenv("TERM")
		os.Unsetenv("COLORTERM")
	}()

	caps := ui.DetectTerminalCapabilities()

	if !caps.HasColor {
		t.Errorf("Expected HasColor to be true")
	}
	if !caps.Supports256Color {
		t.Errorf("Expected Supports256Color to be true")
	}
	if !caps.SupportsTrueColor {
		t.Errorf("Expected SupportsTrueColor to be true")
	}
}

func TestDetectTerminalCapabilitiesDumb(t *testing.T) {
	os.Setenv("TERM", "dumb")
	defer os.Unsetenv("TERM")

	caps := ui.DetectTerminalCapabilities()

	if caps.SupportsMouse {
		t.Errorf("Expected SupportsMouse to be false for dumb terminal")
	}
	if caps.Supports256Color {
		t.Errorf("Expected Supports256Color to be false for dumb terminal")
	}
}

func TestTerminalCapabilitiesFields(t *testing.T) {
	caps := ui.DetectTerminalCapabilities()

	caps.HasColor = true
	caps.Supports256Color = true
	caps.SupportsTrueColor = true
	caps.SupportsMouse = true
	caps.SupportsBracketedPaste = true
	caps.SupportsCursor = true
	caps.SupportsAltScreen = true

	if !caps.HasColor {
		t.Errorf("HasColor field should be settable")
	}
	if !caps.Supports256Color {
		t.Errorf("Supports256Color field should be settable")
	}
	if !caps.SupportsTrueColor {
		t.Errorf("SupportsTrueColor field should be settable")
	}
	if !caps.SupportsMouse {
		t.Errorf("SupportsMouse field should be settable")
	}
	if !caps.SupportsBracketedPaste {
		t.Errorf("SupportsBracketedPaste field should be settable")
	}
	if !caps.SupportsCursor {
		t.Errorf("SupportsCursor field should be settable")
	}
	if !caps.SupportsAltScreen {
		t.Errorf("SupportsAltScreen field should be settable")
	}
}