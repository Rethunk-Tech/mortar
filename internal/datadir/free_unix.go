//go:build unix

package datadir

import (
	"math"

	"golang.org/x/sys/unix"
)

// freeBytes is the space available to an unprivileged writer: bavail blocks of bsize bytes. Each OS types the
// statfs fields differently, so callers convert to int64 first.
func freeBytes(bsize, bavail int64) (int64, error) {
	if bsize <= 0 {
		return 0, unix.EINVAL
	}
	if bavail < 0 {
		return 0, nil
	}
	total := uint64(bavail) * uint64(bsize)
	if total > math.MaxInt64 {
		return math.MaxInt64, nil
	}
	return int64(total), nil
}
