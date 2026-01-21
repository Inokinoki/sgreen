package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents parsed screen configuration
type Config struct {
	Escape          string
	Shell           string
	Scrollback      int
	Logfile         string
	Logging         bool
	FlowControl     string
	Interrupt       bool
	StartupMessage  bool
	Bell            bool
	VBell           bool
	Activity        string
	Silence         string
	Hardstatus      string
	Caption         string
	ShellTitle      string
	Bindings        map[string]string
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Escape:         "^Aa",
		Shell:          "",
		Scrollback:     1000,
		Logfile:        "",
		Logging:        false,
		FlowControl:    "off",
		Interrupt:      false,
		StartupMessage: true,
		Bell:           true,
		VBell:          false,
		Activity:       "",
		Silence:        "",
		Hardstatus:     "",
		Caption:        "",
		ShellTitle:     "",
		Bindings:       make(map[string]string),
	}
}

// FindConfigFile finds the configuration file to use
func FindConfigFile(specifiedFile string) (string, error) {
	// If specified, use it
	if specifiedFile != "" {
		if _, err := os.Stat(specifiedFile); err == nil {
			return specifiedFile, nil
		}
		return "", fmt.Errorf("config file not found: %s", specifiedFile)
	}

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

	// No config file found
	return "", nil
}

// ParseConfigFile parses a screenrc configuration file
func ParseConfigFile(filename string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	processedFiles := make(map[string]bool) // Track processed files to avoid cycles

	return parseConfigLines(lines, config, filepath.Dir(filename), processedFiles)
}

// parseConfigLines parses configuration lines, handling source directives
func parseConfigLines(lines []string, config *Config, baseDir string, processedFiles map[string]bool) (*Config, error) {
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle line continuation (backslash at end of line)
		if strings.HasSuffix(line, "\\") {
			line = strings.TrimSuffix(line, "\\")
			// Merge with next line
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				line = line + " " + nextLine
				// Skip next line in iteration
				continue
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
				// Resolve relative paths
				if !filepath.IsAbs(sourceFile) {
					sourceFile = filepath.Join(baseDir, sourceFile)
				}
				
				// Avoid cycles
				if processedFiles[sourceFile] {
					continue
				}
				processedFiles[sourceFile] = true

				// Parse source file
				sourceData, err := os.ReadFile(sourceFile)
				if err == nil {
					sourceLines := strings.Split(string(sourceData), "\n")
					_, err = parseConfigLines(sourceLines, config, filepath.Dir(sourceFile), processedFiles)
					if err != nil {
						// Non-fatal, continue
					}
				}
			}

		case "escape":
			if len(args) >= 1 {
				config.Escape = args[0]
				if len(args) >= 2 {
					// Escape format: ^Aa (command char + literal char)
					// We'll parse this in the main config loader
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
			if len(args) >= 1 && args[0] == "on" {
				config.VBell = true
			} else {
				config.VBell = false
			}

		case "activity":
			if len(args) >= 1 {
				config.Activity = args[0]
			}

		case "silence":
			if len(args) >= 1 {
				config.Silence = args[0]
			}

		case "hardstatus":
			if len(args) >= 1 {
				config.Hardstatus = strings.Join(args, " ")
			}

		case "caption":
			if len(args) >= 1 {
				config.Caption = strings.Join(args, " ")
			}

		case "shelltitle":
			if len(args) >= 1 {
				config.ShellTitle = strings.Join(args, " ")
			}

		case "bind", "bindkey":
			// bind key command
			// Format: bind key command or bindkey key command
			if len(args) >= 2 {
				key := args[0]
				command := strings.Join(args[1:], " ")
				config.Bindings[key] = command
			}

		case "unbind", "unbindkey":
			// unbind key
			if len(args) >= 1 {
				delete(config.Bindings, args[0])
			}
		}
	}

	return config, nil
}

// ApplyToMainConfig applies parsed config to main config struct
func (c *Config) ApplyToMainConfig(mainConfig interface{}) {
	// This will be called from main.go to apply settings
	// The mainConfig should be a pointer to the main Config struct
}

