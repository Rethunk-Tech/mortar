//go:build !windows

package usererr

func platformDiskFull(error) bool { return false }
