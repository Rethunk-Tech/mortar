//go:build !windows

package cli

// attachConsole is only needed by the Windows GUI-subsystem exe.
func attachConsole() {}
