//go:build !windows

package doctor

import "syscall"

func freeSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	if stat.Bsize < 0 {
		return 0, syscall.EINVAL
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}
