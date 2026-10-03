package datadir

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Tier is one way to put a store file into a profile folder.
type Tier int

const (
	TierClone    Tier = 1
	TierHardlink Tier = 2
	TierSymlink  Tier = 3
	TierCopy     Tier = 4
)

// Ops is the per-file materialiser. Tests replace Tiers and the four funcs.
type Ops struct {
	Tiers                          []Tier
	Clone, Hardlink, Symlink, Copy func(src, dst string) error
	DisableCache                   bool
}

type devCaps struct {
	clone, hardlink, symlink int8
}

var capCache sync.Map // uint64 device -> *devCaps

func defaultOps() Ops {
	return Ops{
		Tiers:    []Tier{TierClone, TierHardlink, TierSymlink, TierCopy},
		Clone:    cloneFile,
		Hardlink: os.Link,
		Symlink: func(src, dst string) error {
			abs, err := filepath.Abs(src)
			if err != nil {
				return err
			}
			return os.Symlink(abs, dst)
		},
		Copy: CopyFile,
	}
}

// MaterializeTree puts src's files into dst using clone, hardlink, symlink, then copy.
func MaterializeTree(src, dst string) error {
	ops := defaultOps()
	return copyTree(src, dst, nil, ops.put)
}

// MaterializeTreeExclusive clones or copies only, so two profiles never share an inode.
func MaterializeTreeExclusive(src, dst string) error {
	ops := defaultOps()
	ops.Tiers = []Tier{TierClone, TierCopy}
	return copyTree(src, dst, nil, ops.put)
}

// MaterializeTreeOps is MaterializeTree with injectable tiers (tests).
func MaterializeTreeOps(src, dst string, ops Ops) error {
	if len(ops.Tiers) == 0 {
		ops.Tiers = defaultOps().Tiers
	}
	if ops.Clone == nil {
		ops.Clone = cloneFile
	}
	if ops.Hardlink == nil {
		ops.Hardlink = os.Link
	}
	if ops.Symlink == nil {
		ops.Symlink = defaultOps().Symlink
	}
	if ops.Copy == nil {
		ops.Copy = CopyFile
	}
	return copyTree(src, dst, nil, ops.put)
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

func (ops Ops) put(src, dst, rel string) error {
	writable := WritableRel(rel) || EditableText(rel)
	srcDev, srcErr := fileDevice(src)
	dstDev, dstErr := fileDevice(filepath.Dir(dst))
	sameDev := srcErr == nil && dstErr == nil && srcDev == dstDev
	var caps *devCaps
	if !ops.DisableCache && dstErr == nil {
		if v, ok := capCache.Load(dstDev); ok {
			c, ok := v.(*devCaps)
			if !ok {
				c = &devCaps{}
				capCache.Store(dstDev, c)
			}
			caps = c
		} else {
			caps = &devCaps{}
			capCache.Store(dstDev, caps)
		}
	}
	for _, t := range ops.Tiers {
		switch t {
		case TierHardlink, TierSymlink:
			if writable {
				continue
			}
		case TierClone, TierCopy:
		default:
			continue
		}
		if !sameDev && (t == TierClone || t == TierHardlink) {
			continue
		}
		if caps != nil {
			switch t {
			case TierClone:
				if caps.clone < 0 {
					continue
				}
			case TierHardlink:
				if caps.hardlink < 0 {
					continue
				}
			case TierSymlink:
				if caps.symlink < 0 {
					continue
				}
			case TierCopy:
			default:
				continue
			}
		}
		fn := ops.fn(t)
		if fn == nil {
			continue
		}
		err := fn(src, dst)
		if err == nil {
			if caps != nil {
				switch t {
				case TierClone:
					caps.clone = 1
				case TierHardlink:
					caps.hardlink = 1
				case TierSymlink:
					caps.symlink = 1
				case TierCopy:
				default:
				}
			}
			return nil
		}
		if !isLinkFallback(err) {
			return err
		}
		if caps != nil && !isCrossDevice(err) {
			switch t {
			case TierClone:
				caps.clone = -1
			case TierHardlink:
				caps.hardlink = -1
			case TierSymlink:
				caps.symlink = -1
			case TierCopy:
			default:
			}
		}
	}
	return CopyFile(src, dst)
}

func (ops Ops) fn(t Tier) func(src, dst string) error {
	switch t {
	case TierClone:
		return ops.Clone
	case TierHardlink:
		return ops.Hardlink
	case TierSymlink:
		return ops.Symlink
	case TierCopy:
		return ops.Copy
	default:
		return nil
	}
}
