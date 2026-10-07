//go:build !windows

package migrate

func fileLocked(error) bool { return false }
