package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inoki/sgreen/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
		return
	}

	if cfg.Escape != "^Aa" {
		t.Errorf("Expected Escape ^Aa, got %s", cfg.Escape)
	}

	if cfg.Scrollback != 1000 {
		t.Errorf("Expected Scrollback 1000, got %d", cfg.Scrollback)
	}

	if cfg.FlowControl != "off" {
		t.Errorf("Expected FlowControl off, got %s", cfg.FlowControl)
	}

	if !cfg.StartupMessage {
		t.Errorf("Expected StartupMessage true")
	}

	if !cfg.Bell {
		t.Errorf("Expected Bell true")
	}
}

func TestFindConfigFileWithSpecified(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.screenrc")

	err := os.WriteFile(configFile, []byte("# test config"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	found, err := config.FindConfigFile(configFile)
	if err != nil {
		t.Errorf("FindConfigFile should find existing file: %v", err)
	}

	if found != configFile {
		t.Errorf("Found path mismatch: got %s, want %s", found, configFile)
	}
}

func TestFindConfigFileNonExistent(t *testing.T) {
	nonExistent := "/non/existent/path/screenrc"

	_, err := config.FindConfigFile(nonExistent)
	if err == nil {
		t.Errorf("FindConfigFile should error for non-existent specified file")
	}
}

func TestFindConfigFileEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "custom.screenrc")

	err := os.WriteFile(configFile, []byte("# test config"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	os.Setenv("SCREENRC", configFile)
	defer os.Unsetenv("SCREENRC")

	found, err := config.FindConfigFile("")
	if err != nil {
		t.Errorf("FindConfigFile should find file via SCREENRC: %v", err)
	}

	if found != configFile {
		t.Errorf("Found path mismatch: got %s, want %s", found, configFile)
	}
}

func TestFindConfigFileHomeDir(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	configFile := filepath.Join(homeDir, ".screenrc")

	err := os.MkdirAll(homeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create home dir: %v", err)
	}

	err = os.WriteFile(configFile, []byte("# test config"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	os.Setenv("HOME", homeDir)
	defer os.Unsetenv("HOME")

	found, err := config.FindConfigFile("")
	if err != nil {
		t.Errorf("FindConfigFile should find file in home dir: %v", err)
	}

	if found != configFile {
		t.Errorf("Found path mismatch: got %s, want %s", found, configFile)
	}
}

func TestFindConfigFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")
	os.Unsetenv("SCREENRC")

	found, err := config.FindConfigFile("")
	if err != nil {
		t.Errorf("FindConfigFile should not error when no config found: %v", err)
	}

	if found != "" {
		t.Errorf("Found path should be empty when no config exists, got %s", found)
	}
}

func TestConfigStructure(t *testing.T) {
	cfg := &config.Config{
		Escape:         "^Zz",
		Shell:          "/bin/zsh",
		Scrollback:     2000,
		Logfile:        "/tmp/sgreen.log",
		Logging:        true,
		FlowControl:    "on",
		Interrupt:      true,
		StartupMessage: false,
		Bell:           false,
		VBell:          true,
		Activity:       "30",
		Silence:        "15",
		Hardstatus:     "always",
		Caption:        "%n %t",
		ShellTitle:     "sgreen",
		Bindings:       map[string]string{"^A": "detach"},
	}

	if len(cfg.Bindings) != 1 {
		t.Errorf("Expected 1 binding, got %d", len(cfg.Bindings))
	}

	binding, exists := cfg.Bindings["^A"]
	if !exists || binding != "detach" {
		t.Errorf("Binding not found or incorrect")
	}
}

func TestConfigEmptyBindings(t *testing.T) {
	cfg := &config.Config{
		Bindings: map[string]string{},
	}

	if cfg.Bindings == nil {
		t.Errorf("Bindings should not be nil")
	}

	if len(cfg.Bindings) != 0 {
		t.Errorf("Expected empty bindings")
	}
}

func TestConfigNilBindings(t *testing.T) {
	cfg := &config.Config{}

	if cfg.Bindings != nil {
		t.Errorf("Bindings should be nil by default")
	}
}
