//go:build unix && !linux

package datasvc

import (
	"os"
	"syscall"
)

func fileAlloc(info os.FileInfo) (dev, ino uint64, allocated int64, ok bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, 0, false
	}
	return uint64(st.Dev), uint64(st.Ino), st.Blocks * 512, true
}
