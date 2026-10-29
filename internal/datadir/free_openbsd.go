//go:build openbsd

package datadir

import "golang.org/x/sys/unix"

func FreeBytes(path string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return freeBytes(st.F_bsize, st.F_bavail)
}
