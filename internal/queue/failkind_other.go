//go:build !windows

package queue

func platformBlocked(error) bool { return false }
