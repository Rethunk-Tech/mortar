//go:build !windows

package fsx

import (
	"errors"
	"os"
	"syscall"
)

// LinkCount is the number of hard links to path itself (a symlink is not followed).
func LinkCount(path string) (uint64, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("no link count for " + path)
	}
	return widen(st.Nlink), nil
}

// widen takes Nlink at whatever width the platform declares it (uint16 on macOS, uint64 on linux/amd64).
func widen[T uint16 | uint32 | uint64](n T) uint64 { return uint64(n) }
