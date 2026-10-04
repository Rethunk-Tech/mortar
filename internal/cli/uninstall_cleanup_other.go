//go:build !windows && !linux

package cli

// On macOS the Steam shortcuts are all it removes.
func removePlatformLeftovers() error { return nil }
