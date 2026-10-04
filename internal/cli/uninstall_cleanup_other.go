//go:build !windows

package cli

// Only the Windows installer has an uninstaller to run this; elsewhere the Steam shortcuts are all it can remove.
func removePlatformLeftovers() error { return nil }
