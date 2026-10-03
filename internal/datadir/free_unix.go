//go:build unix

package datadir

import (
	"math"

	"golang.org/x/sys/unix"
)

func FreeBytes(path string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	if st.Bsize <= 0 {
		return 0, unix.EINVAL
	}
	total := st.Bavail * uint64(st.Bsize)
	if total > math.MaxInt64 {
		return math.MaxInt64, nil
	}
	return int64(total), nil
}
