package datadir

import (
	"errors"
	"fmt"

	"github.com/Rethunk-Tech/go-safefs"
)

// CopyProgress counts files and bytes while a copy runs.
type CopyProgress = safefs.CopyProgress

// copyTempSuffix names every copy's temp file. Deploy's crash recovery removes a leftover of this name in game
// folders, so it is a format 0.3.8 already wrote and must not change.
const copyTempSuffix = ".mortar-tmp"

var mortarCopy = safefs.CopyOptions{TempSuffix: copyTempSuffix}

// CopyTempName is the file CopyFile writes before renaming it to dst; a crash mid-copy leaves it behind.
func CopyTempName(dst string) string { return dst + copyTempSuffix }

// CopyFile copies the regular file src to dst, replacing what is there, so a crash leaves dst whole or absent.
func CopyFile(src, dst string) error { return safefs.CopyFile(src, dst, mortarCopy) }

// CopyTree copies the regular files and folders under src into dst (which may exist). Linked folders are not
// followed, and a symlinked file is copied by content only when it still resolves under src.
func CopyTree(src, dst string) error {
	_, err := safefs.CopyTree(src, dst, mortarCopy)
	return err
}

// CopyTreeResolvingLinks copies src to dst like CopyTree but writes the file a file link points to wherever it lives.
// It is for a folder the user chose on their own disk, such as a game's Mods folder that a symlink-deploying mod
// manager (Vortex) filled with links into its staging folder.
func CopyTreeResolvingLinks(src, dst string) error {
	opts := mortarCopy
	opts.Skip = vortexTag
	err := safefs.CopyTreeFollowingLinks(src, dst, opts)
	if dangling, ok := errors.AsType[*safefs.DanglingLinkError](err); ok {
		return fmt.Errorf("%s links to a file that no longer exists; if Vortex put it there, its staging folder was moved or purged", dangling.Rel)
	}
	return err
}

// vortexTag reports the marker files Vortex leaves in every folder it deploys into
// (LinkingDeployment.ts: "__folder_managed_by_vortex", dot-prefixed off Windows, and the older "__delete_if_empty");
// they are Vortex's bookkeeping, not mod content.
func vortexTag(name string) bool {
	switch name {
	case "__folder_managed_by_vortex", ".__folder_managed_by_vortex", "__delete_if_empty":
		return true
	}
	return false
}
