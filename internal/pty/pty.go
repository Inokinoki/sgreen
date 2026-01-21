package pty

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
)

// PTYProcess represents a PTY process with its command and PTY file
type PTYProcess struct {
	Cmd     *exec.Cmd
	Pty     *os.File
	PtsPath string // Path to the PTY slave device
}

// Start creates a new PTY process with the given command and arguments
func Start(cmdPath string, args []string) (*PTYProcess, error) {
	return StartWithEnv(cmdPath, args, nil)
}

// StartWithEnv creates a new PTY process with custom environment variables
func StartWithEnv(cmdPath string, args []string, envOverrides map[string]string) (*PTYProcess, error) {
	cmd := exec.Command(cmdPath, args...)

	// Set process group management (Unix only)
	setProcessGroup(cmd)

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

	ptyFile, err := pty.Start(cmd)
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

// Pipe connects the client's input/output to the PTY
// It copies data bidirectionally between client and PTY
func (p *PTYProcess) Pipe(clientIn io.Reader, clientOut io.Writer) error {
	// Copy from client input to PTY
	go func() {
		io.Copy(p.Pty, clientIn)
	}()

	// Copy from PTY to client output (main loop)
	_, err := io.Copy(clientOut, p.Pty)
	return err
}

// SetSize sets the size of the PTY
func (p *PTYProcess) SetSize(rows, cols uint16) error {
	if p.Pty == nil {
		return os.ErrInvalid
	}
	return pty.Setsize(p.Pty, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// Close closes the PTY file
func (p *PTYProcess) Close() error {
	if p.Pty != nil {
		return p.Pty.Close()
	}
	return nil
}

// Wait waits for the command to finish
func (p *PTYProcess) Wait() error {
	if p.Cmd != nil {
		return p.Cmd.Wait()
	}
	return nil
}

// Kill kills the underlying process
func (p *PTYProcess) Kill() error {
	if p.Cmd != nil && p.Cmd.Process != nil {
		return p.Cmd.Process.Kill()
	}
	return nil
}
