package datadir

import "github.com/Rethunk-Tech/go-safefs"

// CopyFileNoSync is CopyFile without the fsync; the caller makes the bytes durable with safefs.Flush before it relies
// on them.
func CopyFileNoSync(src, dst string) error { return safefs.CopyFileNoSync(src, dst, mortarCopy) }
