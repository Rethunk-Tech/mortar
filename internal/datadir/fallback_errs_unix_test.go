//go:build !windows

package datadir

import "syscall"

// The errors a clone, hard link and symlink refuse with when the target cannot do them.
var errNoClone, errCrossDevice, errBadLink error = syscall.EOPNOTSUPP, syscall.EXDEV, syscall.EINVAL
