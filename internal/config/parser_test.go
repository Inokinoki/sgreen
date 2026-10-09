package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	tests := []struct {
		name     string
		field    string
		expected interface{}
		actual   interface{}
	}{
		{"Escape", "Escape", "^Aa", cfg.Escape},
		{"Scrollback", "Scrollback", 1000, cfg.Scrollback},
		{"Logging", "Logging", false, cfg.Logging},
		{"StartupMessage", "StartupMessage", true, cfg.StartupMessage},
		{"Bell", "Bell", true, cfg.Bell},
		{"VBell", "VBell", false, cfg.VBell},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expected != tt.actual {
				t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.actual)
			}
		})
	}

	if cfg.Bindings == nil {
		t.Error("Bindings should not be nil")
	}
}

func TestFindConfigFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		wantErr bool
	}{
		{
			name: "specified file exists",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, ".screenrc")
				if err := os.WriteFile(configFile, []byte("# test"), 0644); err != nil {
					t.Fatal(err)
				}
				return configFile, func() {}
			},
			wantErr: false,
		},
		{
			name: "specified file not found",
			setup: func() (string, func()) {
				return "/nonexistent/path/.screenrc", func() {}
			},
			wantErr: true,
		},
		{
			name: "no config file found",
			setup: func() (string, func()) {
				return "", func() {}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specifiedFile, cleanup := tt.setup()
			defer cleanup()

			_, err := FindConfigFile(specifiedFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseConfigFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name:    "simple escape config",
			content: "escape ^Zz\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Escape != "^Zz" {
					t.Errorf("expected escape ^Zz, got %s", cfg.Escape)
				}
			},
		},
		{
			name:    "multiple directives",
			content: "escape ^Zz\ndefscrollback 2000\nshell /bin/zsh\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Escape != "^Zz" {
					t.Errorf("expected escape ^Zz, got %s", cfg.Escape)
				}
				if cfg.Scrollback != 2000 {
					t.Errorf("expected scrollback 2000, got %d", cfg.Scrollback)
				}
				if cfg.Shell != "/bin/zsh" {
					t.Errorf("expected shell /bin/zsh, got %s", cfg.Shell)
				}
			},
		},
		{
			name:    "comments and empty lines",
			content: "# This is a comment\n\nescape ^Zz\n  \n# Another comment\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Escape != "^Zz" {
					t.Errorf("expected escape ^Zz, got %s", cfg.Escape)
				}
			},
		},
		{
			name:    "boolean settings",
			content: "bell off\nvbell on\nlog on\nstartup_message off\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Bell != false {
					t.Errorf("expected bell off, got %v", cfg.Bell)
				}
				if cfg.VBell != true {
					t.Errorf("expected vbell on, got %v", cfg.VBell)
				}
				if cfg.Logging != true {
					t.Errorf("expected log on, got %v", cfg.Logging)
				}
				if cfg.StartupMessage != false {
					t.Errorf("expected startup_message off, got %v", cfg.StartupMessage)
				}
			},
		},
		{
			name:    "bindings",
			content: "bind ^A detach\nbind ^Z kill\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Bindings == nil {
					t.Error("Bindings should not be nil")
				}
				if cfg.Bindings["^A"] != "detach" {
					t.Errorf("expected ^A to bind to detach, got %s", cfg.Bindings["^A"])
				}
				if cfg.Bindings["^Z"] != "kill" {
					t.Errorf("expected ^Z to bind to kill, got %s", cfg.Bindings["^Z"])
				}
			},
		},
		{
			name:    "activity and silence",
			content: "activity 30\nsilence 15\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Activity != "30" {
					t.Errorf("expected activity 30, got %s", cfg.Activity)
				}
				if cfg.Silence != "15" {
					t.Errorf("expected silence 15, got %s", cfg.Silence)
				}
			},
		},
		{
			name:    "hardstatus and caption",
			content: "hardstatus always\ncaption always\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Hardstatus != "always" {
					t.Errorf("expected hardstatus always, got %s", cfg.Hardstatus)
				}
				if cfg.Caption != "always" {
					t.Errorf("expected caption always, got %s", cfg.Caption)
				}
			},
		},
		{
			name:    "logfile",
			content: "logfile /tmp/sgreen.log\n",
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Logfile != "/tmp/sgreen.log" {
					t.Errorf("expected logfile /tmp/sgreen.log, got %s", cfg.Logfile)
				}
				if !cfg.Logging {
					t.Error("expected Logging to be true when logfile is set")
				}
			},
		},
		{
			name:    "invalid file",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configFile string
			if tt.content != "" {
				tmpDir := t.TempDir()
				configFile = filepath.Join(tmpDir, ".screenrc")
				if err := os.WriteFile(configFile, []byte(tt.content), 0644); err != nil {
					t.Fatal(err)
				}
			} else {
				configFile = "/nonexistent/path/.screenrc"
			}

			cfg, err := ParseConfigFile(configFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfigFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestParseConfigLineContinuation(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".screenrc")
	content := "hardstatus always \\\nlastline\ncaption always\n"
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseConfigFile() error = %v", err)
	}

	expected := "always lastline"
	if cfg.Hardstatus != expected {
		t.Errorf("expected hardstatus %q, got %q", expected, cfg.Hardstatus)
	}
}
