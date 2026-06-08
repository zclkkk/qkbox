//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

func windowBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "qkbox-window"
	}
	exeDir := filepath.Dir(exe)

	switch runtime.GOOS {
	case "darwin":
		// App bundle: Contents/MacOS/qkbox → Contents/Helpers/qkbox-window
		helpers := filepath.Join(exeDir, "..", "Helpers", "qkbox-window")
		if _, err := os.Stat(helpers); err == nil {
			return helpers
		}
	default:
		// Linux installed: /usr/lib/qkbox/qkbox-window
		installed := "/usr/lib/qkbox/qkbox-window"
		if _, err := os.Stat(installed); err == nil {
			return installed
		}
	}

	// Dev fallback: same directory as the running executable.
	return filepath.Join(exeDir, "qkbox-window")
}

func prepareDetachedCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
