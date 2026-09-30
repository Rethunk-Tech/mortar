//go:build !windows

package store

func platformDiskFull(error) bool { return false }
