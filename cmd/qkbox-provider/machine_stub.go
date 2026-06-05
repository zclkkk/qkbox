//go:build !windows && !linux

package main

import "runtime"

func machineNetworkAvailable() (bool, string) {
	return false, "Provider-hosted machine network mode is unavailable on " + runtime.GOOS + "."
}
