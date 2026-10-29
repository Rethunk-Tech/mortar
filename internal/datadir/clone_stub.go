//go:build !linux && !windows

package datadir

import (
	"errors"
	"syscall"
)

func cloneFile(string, string) error { return syscall.EOPNOTSUPP }

// The errors arrive wrapped in an *os.LinkError, so they are matched through the chain.
func isLinkFallback(err error) bool {
	return errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.EOPNOTSUPP) || errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.EPERM)
}

func isCrossDevice(err error) bool { return errors.Is(err, syscall.EXDEV) }

func fileDevice(string) (uint64, error) { return 0, syscall.EOPNOTSUPP }
