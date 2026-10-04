//go:build !linux && !windows

package datasvc

import "os"

func fileAlloc(info os.FileInfo) (dev, ino uint64, allocated int64, ok bool) {
	return 0, 0, 0, false
}
