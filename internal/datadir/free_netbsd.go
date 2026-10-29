//go:build netbsd

package datadir

import "golang.org/x/sys/unix"

func FreeBytes(path string) (int64, error) {
	var st unix.Statvfs_t
	if err := unix.Statvfs(path, &st); err != nil {
		return 0, err
	}
	return freeBytes(int64(st.Frsize), int64(st.Bavail))
}
