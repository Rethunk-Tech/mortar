package datadir

import (
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

// materializer shares a store file with a profile folder by the cheapest tier the disk allows, except the files a
// mod or player writes, which never share an inode with the store.
var materializer = safefs.Materializer{
	Shareable: func(rel string) bool { return !WritableRel(rel) && !EditableText(rel) },
	Copy:      mortarCopy,
}

// MaterializeTree puts src's files into dst using clone, hardlink, symlink, then copy.
func MaterializeTree(src, dst string) error { return materializer.Tree(src, dst) }

// MaterializeFile puts one store file at dst the way MaterializeTree would; rel is its path inside the mod, which
// decides whether it may share an inode with the store.
func MaterializeFile(src, dst, rel string) error { return materializer.File(src, dst, rel) }

// MaterializeTreeExclusive clones or copies only, so two profiles never share an inode.
func MaterializeTreeExclusive(src, dst string) error {
	m := materializer
	m.Tiers = []safefs.Tier{safefs.TierClone, safefs.TierCopy}
	return m.Tree(src, dst)
}

// textExts are files mods and players edit in place; a hardlinked or symlinked copy would carry that edit into the
// store and every other profile, so they are cloned or copied. Binaries (dll, png, xnb, audio) stay shareable.
var textExts = map[string]bool{
	".json": true, ".txt": true, ".ini": true, ".cfg": true, ".xml": true, ".yaml": true, ".yml": true,
	".toml": true, ".csv": true, ".tmx": true, ".md": true,
}

// EditableText reports a text file a player may edit in place, so it never shares an inode with the store.
func EditableText(rel string) bool {
	return textExts[strings.ToLower(filepath.Ext(rel))]
}

// WritableRel reports state a mod writes itself (config, data and save folders): never hardlinked or symlinked, and
// owned by the profile rather than compared with the store copy.
func WritableRel(rel string) bool {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	lower := strings.ToLower(base)
	if strings.EqualFold(base, "config.json") || strings.HasSuffix(lower, ".mortar-old") {
		return true
	}
	for p := range strings.SplitSeq(strings.ToLower(rel), "/") {
		switch p {
		case "data", "saves", "save", "savedata":
			return true
		}
	}
	return false
}
