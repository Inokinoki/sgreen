//go:build !windows
// +build !windows

package pty

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/creack/pty"
)

// StartWithEnv creates a new PTY process with custom environment variables
func StartWithEnv(cmdPath string, args []string, envOverrides map[string]string) (*PTYProcess, error) {
	buildCmd := func(withProcessGroup bool) *exec.Cmd {
		cmd := exec.Command(cmdPath, args...)
		if withProcessGroup {
			// Set process group management (Unix only)
			setProcessGroup(cmd)
		}

		// Start with current environment
		cmd.Env = os.Environ()

		// Apply environment overrides
		if envOverrides != nil {
			envMap := make(map[string]string)
			// Parse existing environment
			for _, env := range cmd.Env {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}
			// Apply overrides
			for key, value := range envOverrides {
				envMap[key] = value
			}
			// Rebuild environment slice
			cmd.Env = make([]string, 0, len(envMap))
			for key, value := range envMap {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
		}

		return cmd
	}

	cmd := buildCmd(true)
	ptyFile, err := pty.Start(cmd)
	if err != nil && errors.Is(err, syscall.EPERM) {
		// Some sandboxes deny setpgid; retry without process group management.
		cmd = buildCmd(false)
		ptyFile, err = pty.Start(cmd)
	}
	if err != nil {
		return nil, err
	}

	// Get the PTY slave path
	ptsPath, err := getPtsPath(ptyFile)
	if err != nil {
		// Non-fatal, continue without pts path
		ptsPath = ""
	}

	return &PTYProcess{
		Cmd:     cmd,
		Pty:     ptyFile,
		PtsPath: ptsPath,
	}, nil
}

// getPtsPath gets the path to the PTY slave device
func getPtsPath(ptyFile *os.File) (string, error) {
	if ptyFile == nil {
		return "", os.ErrNotExist
	}

	name := ptyFile.Name()

	// If the name already looks like a pts path, use it
	if filepath.Dir(name) == "/dev/pts" {
		return name, nil
	}

	// Try to read the symlink from /proc/self/fd (Linux)
	if fdPath := filepath.Join("/proc/self/fd", filepath.Base(name)); fdPath != "" {
		if linkPath, err := os.Readlink(fdPath); err == nil {
			if filepath.Dir(linkPath) == "/dev/pts" {
				return linkPath, nil
			}
		}
	}

	// Try using TIOCGPTN ioctl on Unix systems (Linux, BSD)
	ptsPath, err := getPtsPathViaIoctl(ptyFile)
	if err == nil && ptsPath != "" {
		return ptsPath, nil
	}

	// Last resort: return empty string (non-fatal)
	return "", os.ErrNotExist
}

// setLocalSize applies a window size to the local PTY master.
func (p *PTYProcess) setLocalSize(rows, cols uint16) error {
	if p.Pty == nil {
		return os.ErrInvalid
	}
	return pty.Setsize(p.Pty, &pty.Winsize{Rows: rows, Cols: cols})
}
