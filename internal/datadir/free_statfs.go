//go:build unix && !openbsd && !netbsd

package datadir

import "golang.org/x/sys/unix"

func FreeBytes(path string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return freeBytes(int64(st.Bsize), int64(st.Bavail))
}
