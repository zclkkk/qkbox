//go:build !windows

package main

import "os/exec"

func prepareDetachedCmd(cmd *exec.Cmd) {
	// Unix needs no extra launch flags for the GUI-to-daemon path.
}
