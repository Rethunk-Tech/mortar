//go:build darwin

package datadir

import (
	"errors"

	"golang.org/x/sys/unix"
)

func cloneFile(src, dst string) error {
	return unix.Clonefile(src, dst, unix.CLONE_NOFOLLOW)
}

func isLinkFallback(err error) bool {
	return errors.Is(err, unix.EXDEV) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EINVAL) ||
		errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.ENOSYS)
}

func isCrossDevice(err error) bool {
	return errors.Is(err, unix.EXDEV)
}

func fileDevice(path string) (uint64, error) {
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}
