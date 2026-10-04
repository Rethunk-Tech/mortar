//go:build !linux && !windows

package datadir

import "syscall"

func cloneFile(string, string) error { return syscall.EOPNOTSUPP }

func isLinkFallback(err error) bool {
	return err == syscall.EXDEV || err == syscall.EOPNOTSUPP || err == syscall.EINVAL || err == syscall.EPERM
}

func isCrossDevice(err error) bool { return err == syscall.EXDEV }

func fileDevice(string) (uint64, error) { return 0, syscall.EOPNOTSUPP }
