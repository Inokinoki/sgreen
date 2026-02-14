//go:build windows
// +build windows

package main

import "os/exec"

func setDetachSysProcAttr(cmd *exec.Cmd) {}
