//go:build windows

package main

import "golang.org/x/sys/windows"

func machineNetworkAvailable() (bool, string) {
	token := windows.GetCurrentProcessToken()
	if !token.IsElevated() {
		return false, "Provider must run elevated for Windows machine network mode."
	}
	return true, ""
}
