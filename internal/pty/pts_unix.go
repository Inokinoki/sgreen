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

// getPtsPathViaIoctl resolves the slave device path for a PTY master fd.
// Linux and macOS need different ioctls:
//   - Linux: TIOCGPTN returns the pts number ("/dev/pts" is the devpts root).
//   - macOS/BSD: TIOCPTYGNAME returns the full path in a 128-byte buffer.
//
// On other Unixes this fails gracefully and the caller continues without
// a pts path.
func getPtsPathViaIoctl(ptyFile *os.File) (string, error) {
	ptsPath := getPtsPathLinux(ptyFile)
	if ptsPath != "" {
		if _, err := os.Stat(ptsPath); err == nil {
			return ptsPath, nil
		}
		return "", os.ErrNotExist
	}

	ptsPath = getPtsPathDarwin(ptyFile)
	if ptsPath != "" {
		if _, err := os.Stat(ptsPath); err == nil {
			return ptsPath, nil
		}
		return "", os.ErrNotExist
	}

	return "", os.ErrNotExist
}

// getPtsPathLinux uses TIOCGPTN (0x80045430); the number is baked in
// because the constant is not exported portably by x/sys/unix.
func getPtsPathLinux(ptyFile *os.File) string {
	var ptyNum uint32
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		ptyFile.Fd(),
		uintptr(0x80045430), // TIOCGPTN
		uintptr(unsafe.Pointer(&ptyNum)),
	)
	if errno != 0 {
		return ""
	}
	return filepath.Join("/dev/pts", strconv.FormatUint(uint64(ptyNum), 10))
}

// getPtsPathDarwin uses TIOCPTYGNAME with a 128-byte buffer as the
// kernel requires.
func getPtsPathDarwin(ptyFile *os.File) string {
	const tiocptygname = 0x40807453
	buf := make([]byte, 128)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		ptyFile.Fd(),
		uintptr(tiocptygname),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 {
		return ""
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return ""
}
