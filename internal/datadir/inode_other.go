//go:build !unix && !windows

package datadir

import "os"

// linkedKey reports no sharing: a hard-linked file is copied once per name.
func linkedKey(string, os.FileInfo) (fileKey, bool) { return fileKey{}, false }
