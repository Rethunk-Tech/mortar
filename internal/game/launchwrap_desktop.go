//go:build !server

package game

// launchWrapper is empty in a desktop build: only a server build's self-test sandbox wraps its launches.
func launchWrapper() string { return "" }
