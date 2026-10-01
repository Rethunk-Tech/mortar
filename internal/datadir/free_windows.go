//go:build windows

package datadir

import (
	"golang.org/x/sys/windows"
)

func freeBytes(path string) (int64, error) {
	var free, total, totalFree uint64
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	if err := windows.GetDiskFreeSpaceEx(p, &free, &total, &totalFree); err != nil {
		return 0, err
	}
	return int64(free), nil
}
