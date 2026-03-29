package unit

import (
	"bytes"
	"testing"

	"github.com/inoki/sgreen/internal/ui"
)

func TestShowStartupMessage(t *testing.T) {
	var buf bytes.Buffer
	ui.ShowStartupMessage(&buf, "test_session", 3)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowStartupMessage should produce output")
	}

	contains := func(substr string) bool {
		return bytes.Contains(buf.Bytes(), []byte(substr))
	}

	if !contains("Welcome to sgreen") {
		t.Errorf("Output should contain welcome message")
	}
	if !contains("test_session") {
		t.Errorf("Output should contain session name")
	}
	if !contains("Windows: 3") {
		t.Errorf("Output should contain window count")
	}
}

func TestShowMessage(t *testing.T) {
	var buf bytes.Buffer
	testMsg := "Test message"
	ui.ShowMessage(&buf, testMsg)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowMessage should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte(testMsg)) {
		t.Errorf("Output should contain the message")
	}
}

func TestShowActivityMessage(t *testing.T) {
	var buf bytes.Buffer
	windowTitle := "Test Window"
	ui.ShowActivityMessage(&buf, windowTitle)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowActivityMessage should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("Activity in window")) {
		t.Errorf("Output should contain activity message")
	}
	if !bytes.Contains(buf.Bytes(), []byte(windowTitle)) {
		t.Errorf("Output should contain window title")
	}
}

func TestShowSilenceMessage(t *testing.T) {
	var buf bytes.Buffer
	windowTitle := "Test Window"
	ui.ShowSilenceMessage(&buf, windowTitle)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowSilenceMessage should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("Silence in window")) {
		t.Errorf("Output should contain silence message")
	}
	if !bytes.Contains(buf.Bytes(), []byte(windowTitle)) {
		t.Errorf("Output should contain window title")
	}
}

func TestShowVersion(t *testing.T) {
	var buf bytes.Buffer
	ui.ShowVersion(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowVersion should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("sgreen version")) {
		t.Errorf("Output should contain version info")
	}
	if !bytes.Contains(buf.Bytes(), []byte("terminal multiplexer")) {
		t.Errorf("Output should contain description")
	}
}

func TestShowLicense(t *testing.T) {
	var buf bytes.Buffer
	ui.ShowLicense(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowLicense should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("License")) {
		t.Errorf("Output should contain license info")
	}
}

func TestBlankScreen(t *testing.T) {
	var buf bytes.Buffer
	ui.BlankScreen(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("BlankScreen should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("\033[2J")) {
		t.Errorf("Output should contain clear screen sequence")
	}
}

func TestShowTimeLoad(t *testing.T) {
	var buf bytes.Buffer
	ui.ShowTimeLoad(&buf)

	result := buf.String()
	if result == "" {
		t.Errorf("ShowTimeLoad should produce output")
	}

	if !bytes.Contains(buf.Bytes(), []byte("Time:")) {
		t.Errorf("Output should contain time info")
	}
}

func TestShowBell(t *testing.T) {
	tests := []struct {
		name   string
		visual bool
	}{
		{"audible bell", false},
		{"visual bell", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			ui.ShowBell(&buf, tt.visual)

			if tt.visual {
				if !bytes.Contains(buf.Bytes(), []byte("\033[?5h")) {
					t.Errorf("Visual bell should contain reverse video sequence")
				}
			} else {
				if !bytes.Contains(buf.Bytes(), []byte("\a")) {
					t.Errorf("Audible bell should contain bell character")
				}
			}
		})
	}
}