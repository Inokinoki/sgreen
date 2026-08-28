//go:build !windows
// +build !windows

package pty

import (
	"os"
	"path/filepath"
	"strconv"
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

	// ptyNum is the pts device number returned by the kernel; the master fd's
	// own name ("ptmx") is NOT the pts device.
	ptsPath := filepath.Join("/dev/pts", strconv.FormatUint(uint64(ptyNum), 10))
	if _, err := os.Stat(ptsPath); err == nil {
		return ptsPath, nil
	}

	return "", os.ErrNotExist
}
