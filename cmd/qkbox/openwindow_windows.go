//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const (
	detachedProcess        = 0x00000008
	createBreakawayFromJob = 0x01000000
)

func windowBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "qkbox-window.exe"
	}
	return filepath.Join(filepath.Dir(exe), "qkbox-window.exe")
}

func prepareDetachedCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess | createBreakawayFromJob,
	}
}
