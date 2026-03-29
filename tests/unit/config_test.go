package unit

import (
	"testing"

	"github.com/inoki/sgreen/internal/config"
)

func TestDefaultConfigFields(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg == nil {
		t.Fatalf("DefaultConfig should not return nil")
	}
	if cfg.Escape != "^Aa" {
		t.Errorf("Expected Escape ^Aa, got %s", cfg.Escape)
	}
	if cfg.Scrollback != 1000 {
		t.Errorf("Expected Scrollback 1000, got %d", cfg.Scrollback)
	}
	if cfg.Bell != true {
		t.Errorf("Expected Bell true, got %v", cfg.Bell)
	}
}

func TestConfigFields(t *testing.T) {
	cfg := &config.Config{
		Escape:         "^Zz",
		Shell:          "/bin/zsh",
		Scrollback:     2000,
		Logfile:        "/tmp/test.log",
		Logging:        true,
		FlowControl:    "on",
		Interrupt:      true,
		StartupMessage: false,
		Bell:           false,
		VBell:          true,
		Activity:       "30",
		Silence:        "15",
		Hardstatus:     "always",
		Caption:        "test",
		ShellTitle:     "Test Window",
		Bindings:       map[string]string{"^A": "detach", "^Z": "kill"},
	}

	if cfg.Escape != "^Zz" {
		t.Errorf("Expected Escape ^Zz, got %s", cfg.Escape)
	}
	if cfg.Shell != "/bin/zsh" {
		t.Errorf("Expected Shell /bin/zsh, got %s", cfg.Shell)
	}
	if cfg.Scrollback != 2000 {
		t.Errorf("Expected Scrollback 2000, got %d", cfg.Scrollback)
	}
	if cfg.Logfile != "/tmp/test.log" {
		t.Errorf("Expected Logfile /tmp/test.log, got %s", cfg.Logfile)
	}
	if cfg.Logging != true {
		t.Errorf("Expected Logging true, got %v", cfg.Logging)
	}
	if cfg.FlowControl != "on" {
		t.Errorf("Expected FlowControl on, got %s", cfg.FlowControl)
	}
	if cfg.Interrupt != true {
		t.Errorf("Expected Interrupt true, got %v", cfg.Interrupt)
	}
	if cfg.StartupMessage != false {
		t.Errorf("Expected StartupMessage false, got %v", cfg.StartupMessage)
	}
	if cfg.Bell != false {
		t.Errorf("Expected Bell false, got %v", cfg.Bell)
	}
	if cfg.VBell != true {
		t.Errorf("Expected VBell true, got %v", cfg.VBell)
	}
	if cfg.Activity != "30" {
		t.Errorf("Expected Activity 30, got %s", cfg.Activity)
	}
	if cfg.Silence != "15" {
		t.Errorf("Expected Silence 15, got %s", cfg.Silence)
	}
	if cfg.Hardstatus != "always" {
		t.Errorf("Expected Hardstatus always, got %s", cfg.Hardstatus)
	}
	if cfg.Caption != "test" {
		t.Errorf("Expected Caption test, got %s", cfg.Caption)
	}
	if cfg.ShellTitle != "Test Window" {
		t.Errorf("Expected ShellTitle Test Window, got %s", cfg.ShellTitle)
	}
	if len(cfg.Bindings) != 2 {
		t.Errorf("Expected 2 bindings, got %d", len(cfg.Bindings))
	}
}

func TestConfigEmptyValues(t *testing.T) {
	cfg := &config.Config{
		Escape:         "",
		Shell:          "",
		Scrollback:     0,
		Logfile:        "",
		Logging:        false,
		FlowControl:    "",
		Interrupt:      false,
		StartupMessage: false,
		Bell:           false,
		VBell:          false,
		Activity:       "",
		Silence:        "",
		Hardstatus:     "",
		Caption:        "",
		ShellTitle:     "",
		Bindings:       map[string]string{},
	}

	if cfg.Escape != "" {
		t.Errorf("Expected empty Escape, got %s", cfg.Escape)
	}
	if cfg.Shell != "" {
		t.Errorf("Expected empty Shell, got %s", cfg.Shell)
	}
	if cfg.Scrollback != 0 {
		t.Errorf("Expected Scrollback 0, got %d", cfg.Scrollback)
	}
	if cfg.Logfile != "" {
		t.Errorf("Expected empty Logfile, got %s", cfg.Logfile)
	}
	if cfg.Logging != false {
		t.Errorf("Expected Logging false, got %v", cfg.Logging)
	}
	if cfg.FlowControl != "" {
		t.Errorf("Expected empty FlowControl, got %s", cfg.FlowControl)
	}
	if cfg.Interrupt != false {
		t.Errorf("Expected Interrupt false, got %v", cfg.Interrupt)
	}
	if cfg.StartupMessage != false {
		t.Errorf("Expected StartupMessage false, got %v", cfg.StartupMessage)
	}
	if cfg.Bell != false {
		t.Errorf("Expected Bell false, got %v", cfg.Bell)
	}
	if cfg.VBell != false {
		t.Errorf("Expected VBell false, got %v", cfg.VBell)
	}
	if cfg.Activity != "" {
		t.Errorf("Expected empty Activity, got %s", cfg.Activity)
	}
	if cfg.Silence != "" {
		t.Errorf("Expected empty Silence, got %s", cfg.Silence)
	}
	if cfg.Hardstatus != "" {
		t.Errorf("Expected empty Hardstatus, got %s", cfg.Hardstatus)
	}
	if cfg.Caption != "" {
		t.Errorf("Expected empty Caption, got %s", cfg.Caption)
	}
	if cfg.ShellTitle != "" {
		t.Errorf("Expected empty ShellTitle, got %s", cfg.ShellTitle)
	}
	if len(cfg.Bindings) != 0 {
		t.Errorf("Expected empty Bindings, got %d", len(cfg.Bindings))
	}
}

func TestConfigBindings(t *testing.T) {
	tests := []struct {
		name     string
		bindings map[string]string
		count    int
	}{
		{
			name:     "single binding",
			bindings: map[string]string{"^A": "detach"},
			count:    1,
		},
		{
			name:     "multiple bindings",
			bindings: map[string]string{"^A": "detach", "^Z": "kill", "^D": "detach"},
			count:    3,
		},
		{
			name:     "empty bindings",
			bindings: map[string]string{},
			count:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Bindings: tt.bindings}
			if len(cfg.Bindings) != tt.count {
				t.Errorf("Expected %d bindings, got %d", tt.count, len(cfg.Bindings))
			}
		})
	}
}

func TestConfigNumericFields(t *testing.T) {
	tests := []struct {
		name       string
		scrollback int
		valid      bool
	}{
		{"positive scrollback", 1000, true},
		{"zero scrollback", 0, true},
		{"negative scrollback", -100, true},
		{"large scrollback", 100000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Scrollback: tt.scrollback}
			if cfg.Scrollback != tt.scrollback {
				t.Errorf("Expected Scrollback %d, got %d", tt.scrollback, cfg.Scrollback)
			}
		})
	}
}
