//go:build unix

package datadir

import (
	"os"
	"syscall"
)

// linkedKey returns the inode key of a file with more than one hard link.
func linkedKey(info os.FileInfo) (fileKey, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Nlink < 2 {
		return fileKey{}, false
	}
	return fileKey{u64(st.Dev), u64(st.Ino)}, true
}

// u64 widens a stat field whose width differs per platform.
func u64[T ~uint32 | ~uint64 | ~int32 | ~int64](v T) uint64 { return uint64(v) }
