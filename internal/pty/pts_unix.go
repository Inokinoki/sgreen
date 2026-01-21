//go:build !windows
// +build !windows

package pty

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// getPtsPathViaIoctl attempts to get the pts path using ioctl (Unix-specific)
func getPtsPathViaIoctl(ptyFile *os.File) (string, error) {
	// TIOCGPTN is Linux-specific (0x80045430)
	// On other systems, this will fail gracefully
	var ptyNum uint32
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		ptyFile.Fd(),
		uintptr(0x80045430), // TIOCGPTN
		uintptr(unsafe.Pointer(&ptyNum)),
	)
	if errno != 0 {
		return "", errno
	}

	ptsPath := filepath.Join("/dev/pts", filepath.Base(ptyFile.Name()))
	if _, err := os.Stat(ptsPath); err == nil {
		return ptsPath, nil
	}

	return "", os.ErrNotExist
}
