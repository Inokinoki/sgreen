package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inoki/sgreen/internal/config"
)

// TestFindConfigFileViaEnvVar verifies $SCREENRC resolution (migrated from
// tests/unit; the in-package table test does not cover env-var lookup).
func TestFindConfigFileViaEnvVar(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "custom.screenrc")
	if err := os.WriteFile(configFile, []byte("# test config"), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}
	t.Setenv("SCREENRC", configFile)

	found, err := config.FindConfigFile("")
	if err != nil {
		t.Fatalf("FindConfigFile should find file via SCREENRC: %v", err)
	}
	if found != configFile {
		t.Errorf("found path mismatch: got %s, want %s", found, configFile)
	}
}

// TestFindConfigFileViaHomeDir verifies the $HOME/.screenrc fallback
// (migrated from tests/unit).
func TestFindConfigFileViaHomeDir(t *testing.T) {
	homeDir := filepath.Join(t.TempDir(), "home")
	configFile := filepath.Join(homeDir, ".screenrc")
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("failed to create home dir: %v", err)
	}
	if err := os.WriteFile(configFile, []byte("# test config"), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}
	t.Setenv("HOME", homeDir)
	// os.UserHomeDir reads USERPROFILE (not HOME) on Windows.
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("SCREENRC", "")

	found, err := config.FindConfigFile("")
	if err != nil {
		t.Fatalf("FindConfigFile should find file in home dir: %v", err)
	}
	if found != configFile {
		t.Errorf("found path mismatch: got %s, want %s", found, configFile)
	}
}
