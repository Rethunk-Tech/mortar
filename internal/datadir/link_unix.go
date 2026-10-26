//go:build !windows

package datadir

import "os"

// LinkDir makes link point at the folder target.
func LinkDir(link, target string) error { return os.Symlink(target, link) }
