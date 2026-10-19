//go:build unix

package secret

import (
	"io/fs"
	"os"
	"syscall"
)

// ownedByUser reports whether info belongs to the user running Mortar.
func ownedByUser(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}
