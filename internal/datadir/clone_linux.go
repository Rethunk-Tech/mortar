//go:build linux

package datadir

import (
	"errors"
	"os"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"golang.org/x/sys/unix"
)

func cloneFile(src, dst string) error {
	in, err := fsx.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := fsx.CreateExcl(dst, 0o600)
	if err != nil {
		return err
	}
	if err := unix.IoctlFileClone(int(out.Fd()), int(in.Fd())); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

func isLinkFallback(err error) bool {
	return errors.Is(err, unix.EXDEV) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.EINVAL) ||
		errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.ENOTTY)
}

func isCrossDevice(err error) bool {
	return errors.Is(err, unix.EXDEV)
}

func fileDevice(path string) (uint64, error) {
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return 0, err
	}
	return st.Dev, nil
}
