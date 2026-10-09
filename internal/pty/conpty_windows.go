//go:build windows
// +build windows

package pty

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32                           = windows.NewLazySystemDLL("kernel32.dll")
	procInitializeProcThreadAttributeList = modkernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute         = modkernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList     = modkernel32.NewProc("DeleteProcThreadAttributeList")
)

// StartWithEnv creates a new ConPTY-backed process with custom environment
// variables. The pseudo console owns two one-way pipes: Pty is the side we
// write user input to, PtyRead the side we read program output from.
func StartWithEnv(cmdPath string, args []string, envOverrides map[string]string) (*PTYProcess, error) {
	if cmdPath == "" {
		return nil, fmt.Errorf("empty command")
	}

	var conInR, conInW, conOutR, conOutW windows.Handle
	if err := windows.CreatePipe(&conInR, &conInW, nil, 0); err != nil {
		return nil, fmt.Errorf("create input pipe: %w", err)
	}
	if err := windows.CreatePipe(&conOutR, &conOutW, nil, 0); err != nil {
		windows.CloseHandle(conInR)
		windows.CloseHandle(conInW)
		return nil, fmt.Errorf("create output pipe: %w", err)
	}

	size := windows.Coord{X: 80, Y: 24}
	var pc windows.Handle
	if err := windows.CreatePseudoConsole(size, conInR, conOutW, 0, &pc); err != nil {
		windows.CloseHandle(conInR)
		windows.CloseHandle(conInW)
		windows.CloseHandle(conOutR)
		windows.CloseHandle(conOutW)
		return nil, fmt.Errorf("CreatePseudoConsole: %w", err)
	}
	// CreatePseudoConsole duplicates what it needs; our references to the
	// conpty-side ends must go away or the console never sees EOF.
	windows.CloseHandle(conInR)
	windows.CloseHandle(conOutW)

	proc, err := spawnInPseudoConsole(pc, cmdPath, args, envOverrides)
	if err != nil {
		windows.ClosePseudoConsole(pc)
		windows.CloseHandle(conInW)
		windows.CloseHandle(conOutR)
		return nil, err
	}

	return &PTYProcess{
		Cmd:     proc,
		Pty:     os.NewFile(uintptr(conInW), "conpty-in"),
		PtyRead: os.NewFile(uintptr(conOutR), "conpty-out"),
		conptyResizer: func(rows, cols uint16) error {
			return windows.ResizePseudoConsole(pc, windows.Coord{X: int16(cols), Y: int16(rows)})
		},
	}, nil
}

// spawnInPseudoConsole runs CreateProcessW with the pseudo console attached
// through a startup attribute list.
func spawnInPseudoConsole(pc windows.Handle, cmdPath string, args []string, envOverrides map[string]string) (*exec.Cmd, error) {
	cmd := exec.Command(cmdPath, args...)
	cmd.Env = buildEnv(os.Environ(), envOverrides)

	parts := make([]string, 0, len(cmd.Args))
	for _, a := range cmd.Args {
		parts = append(parts, escapeWindowsArg(a))
	}
	cmdline, err := windows.UTF16PtrFromString(strings.Join(parts, " "))
	if err != nil {
		return nil, err
	}

	envBlock := utf16.Encode([]rune(strings.Join(cmd.Env, "\x00") + "\x00\x00"))

	attrList, attrBuf, err := newPseudoConsoleAttrList(pc)
	if err != nil {
		return nil, err
	}
	defer procDeleteProcThreadAttributeList.Call(uintptr(attrList))

	siex := &windows.StartupInfoEx{}
	siex.Cb = uint32(unsafe.Sizeof(*siex))
	siex.ProcThreadAttributeList = (*windows.ProcThreadAttributeList)(attrList)

	var pi windows.ProcessInformation
	if err := windows.CreateProcess(
		nil, cmdline, nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		&envBlock[0], nil, &siex.StartupInfo, &pi); err != nil {
		return nil, fmt.Errorf("CreateProcess: %w", err)
	}
	windows.CloseHandle(pi.Thread)

	_ = attrBuf
	// Hand the pid to exec.Cmd via FindProcess (lazy handle); the handle
	// from CreateProcess itself is released here.
	windows.CloseHandle(pi.Process)
	found, err := os.FindProcess(int(pi.ProcessId))
	if err != nil {
		return nil, err
	}
	cmd.Process = found
	return cmd, nil
}

// newPseudoConsoleAttrList builds a PROC_THREAD_ATTRIBUTE_LIST carrying
// the pseudo console handle. It returns the list pointer and the backing
// buffer, which must stay alive for the CreateProcess call.
func newPseudoConsoleAttrList(pc windows.Handle) (unsafe.Pointer, []byte, error) {
	var size uintptr
	r, _, _ := procInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	if r == 0 && size == 0 {
		return nil, nil, fmt.Errorf("InitializeProcThreadAttributeList size probe failed")
	}
	buf := make([]byte, size)
	list := unsafe.Pointer(&buf[0])
	r, _, e := procInitializeProcThreadAttributeList.Call(uintptr(list), 1, 0, uintptr(unsafe.Pointer(&size)))
	if r == 0 {
		return nil, nil, fmt.Errorf("InitializeProcThreadAttributeList: %w", e)
	}
	const procThreadAttributePseudoConsole = 0x00020016
	r, _, e = procUpdateProcThreadAttribute.Call(
		uintptr(list), 0,
		procThreadAttributePseudoConsole,
		uintptr(pc), unsafe.Sizeof(pc),
		0, 0)
	if r == 0 {
		return nil, nil, fmt.Errorf("UpdateProcThreadAttribute: %w", e)
	}
	return list, buf, nil
}

// escapeWindowsArg quotes an argv element for a CreateProcessW command
// line (syscall.escapeArg is not exported).
func escapeWindowsArg(s string) string {
	if s == "" {
		return `""`
	}
	needsQuotes := strings.ContainsAny(s, " 	")
	if !needsQuotes {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' || c == 0x5C {
			b.WriteByte(0x5C)
		}
		b.WriteByte(c)
	}
	b.WriteByte('"')
	return b.String()
}

// buildEnv merges overrides into base (key=value strings), preserving order.
func buildEnv(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}
	merged := make(map[string]string, len(base)+len(overrides))
	order := make([]string, 0, len(base)+len(overrides))
	for _, kv := range base {
		if i := strings.IndexByte(kv, '='); i > 0 {
			k := kv[:i]
			if _, dup := merged[k]; !dup {
				order = append(order, k)
			}
			merged[k] = kv[i+1:]
		}
	}
	for k, v := range overrides {
		if _, exists := merged[k]; !exists {
			order = append(order, k)
		}
		merged[k] = v
	}
	out := make([]string, 0, len(order))
	for _, k := range order {
		out = append(out, k+"="+merged[k])
	}
	return out
}

// setLocalSize is unreachable on Windows (SetSize routes to
// conptyResizer); kept for the interface contract.
func (p *PTYProcess) setLocalSize(rows, cols uint16) error {
	return os.ErrInvalid
}
