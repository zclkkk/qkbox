//go:build linux

package main

import "os"

func machineNetworkAvailable() (bool, string) {
	if os.Geteuid() != 0 {
		return false, "Provider must run as root for Linux machine network mode."
	}
	return true, ""
}
