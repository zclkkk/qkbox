package ipcframework

import "crypto/subtle"

func TokenMatches(token, expected string) bool {
	return token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}
