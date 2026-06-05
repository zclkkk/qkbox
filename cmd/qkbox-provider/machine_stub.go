//go:build !windows

package main

import "runtime"

func machineNetworkAvailable() (bool, string) {
	return false, "Windows provider-hosted machine network mode is unavailable on " + runtime.GOOS + "."
}
