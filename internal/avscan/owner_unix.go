//go:build unix

package avscan

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// trustedSocketOwner reports whether a default-list socket is owned by root or the clamav user, so another local
// user who binds a world-writable path (/tmp) cannot answer "OK" for every scan. A path that cannot be stat'ed is
// left for the dial to reject.
func trustedSocketOwner(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return true
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	if st.Uid == 0 {
		return true
	}
	u, err := user.Lookup("clamav")
	if err != nil {
		return false
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	return err == nil && uint32(uid) == st.Uid
}
