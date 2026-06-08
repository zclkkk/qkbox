package main

import "os/exec"

// spawnQKBoxWindow launches the qkbox-window helper binary.
// The binary must be co-located with qkbox (see windowBinaryPath).
func spawnQKBoxWindow() error {
	cmd := exec.Command(windowBinaryPath())
	prepareDetachedCmd(cmd)
	return cmd.Start()
}
