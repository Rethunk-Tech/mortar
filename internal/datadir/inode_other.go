//go:build !unix

package datadir

import "os"

// linkedKey reports no sharing: off unix a hard-linked file is copied once per name.
func linkedKey(os.FileInfo) (fileKey, bool) { return fileKey{}, false }
