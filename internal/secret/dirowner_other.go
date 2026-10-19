//go:build !unix

package secret

import "io/fs"

// ownedByUser is true off Unix: the Flatpak store only exists on Linux.
func ownedByUser(fs.FileInfo) bool { return true }
