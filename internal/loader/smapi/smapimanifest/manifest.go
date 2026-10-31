// Package smapimanifest reads SMAPI mod manifests the way SMAPI does: leniently.
package smapimanifest

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-smapi"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// FileName is the manifest file SMAPI looks for.
const FileName = smapi.FileName

// Manifest holds the fields Mortar uses.
type Manifest struct {
	Name        string
	Author      string
	Version     string
	UniqueID    string
	Description string
	UpdateKeys  []string
	// EntryDll is the C# mod's assembly file, relative to the mod folder; empty for content packs.
	EntryDll string
	// ContentPackFor is the ContentPackFor framework UniqueID when the manifest declares one.
	ContentPackFor string
	// Dependencies lists Dependencies[] and, as a required entry, the ContentPackFor framework.
	Dependencies []mod.Dependency
	// UpdateCautionMessage is shown at update time when Stardrop or Mortar cannot read the new version's manifest yet.
	UpdateCautionMessage string
	// MinimumGameVersion is the oldest game version the mod runs on, when it declares one.
	MinimumGameVersion string
	// DeleteOldVersion tells Mortar to carry over only user-written config (such as config.json), not other edited files.
	DeleteOldVersion bool
	// Format is the mod.ID format UniqueID belongs to; empty is SMAPI, the only format a manifest.json is read in.
	Format string
}

// parsed holds manifests already parsed, by their exact bytes, so the same file read again (a start reads each
// installed manifest several times) is not parsed again. It is emptied when full rather than tracking age.
var parsed struct {
	sync.Mutex
	m map[string]Manifest
}

const maxParsed = 8192

// Parse reads a manifest as leniently as SMAPI does (smapi.Parse). A manifest without a UniqueID is an error, since
// nothing can refer to that mod.
func Parse(b []byte) (Manifest, error) {
	parsed.Lock()
	m, ok := parsed.m[string(b)]
	parsed.Unlock()
	if ok {
		return m.clone(), nil
	}
	sm, err := smapi.Parse(b)
	if err != nil {
		return Manifest{}, err
	}
	m = Manifest{
		Name: sm.Name, Author: sm.Author, Version: sm.Version, UniqueID: sm.UniqueID, Description: sm.Description,
		UpdateKeys: sm.UpdateKeys, EntryDll: sm.EntryDll, ContentPackFor: sm.ContentPackFor,
		UpdateCautionMessage: sm.UpdateCautionMessage, MinimumGameVersion: sm.MinimumGameVersion,
		DeleteOldVersion: sm.DeleteOldVersion,
	}
	for _, d := range sm.Dependencies {
		m.Dependencies = append(m.Dependencies, mod.Dependency{UniqueID: d.UniqueID, MinimumVersion: d.MinimumVersion, Required: d.IsRequired})
	}
	parsed.Lock()
	if parsed.m == nil || len(parsed.m) >= maxParsed {
		parsed.m = map[string]Manifest{}
	}
	parsed.m[string(b)] = m.clone()
	parsed.Unlock()
	return m, nil
}

func (m Manifest) clone() Manifest {
	m.UpdateKeys = slices.Clone(m.UpdateKeys)
	m.Dependencies = slices.Clone(m.Dependencies)
	return m
}

// ReadFile reads dir's manifest.json. Mod managers that deploy by symlink (Vortex, on Linux and on Windows when set
// to) leave it as a link into their own staging folder, so the link is resolved first and the target read through its
// own folder; a dangling link reads as not existing.
func ReadFile(dir string) ([]byte, error) {
	resolved, err := safefs.EvalSymlinks(filepath.Join(dir, FileName))
	if err != nil {
		return nil, err
	}
	return safefs.ReadFile(resolved)
}

// Mod is a manifest found by Scan.
type Mod struct {
	Manifest
	// Folder is the manifest's folder relative to the scanned root, slash-separated; "." for the root itself.
	Folder string
}

// Invalid is a manifest.json that exists but cannot be read as a SMAPI manifest.
type Invalid struct {
	// Folder is the manifest's folder relative to the scanned root, slash-separated; "." for the root itself.
	Folder string
	Err    error
}

// Error names the manifest by its path under the scanned root and says why it was refused.
func (i Invalid) Error() string {
	name := FileName
	if i.Folder != "." {
		name = i.Folder + "/" + FileName
	}
	return name + ": " + i.Err.Error()
}

// Scan finds mods under root like SMAPI: it stops descending at a folder holding a manifest.json and skips
// subfolders whose names start with a dot. A manifest that does not parse is skipped, as SMAPI reports it
// as invalid rather than loading it. Unlike SMAPI it also descends into a folder that holds other files (a stray
// config.json or readme.pdf beside a mod folder): every caller installs each mod found as its own folder, so the
// mod loads once Mortar deploys it, where SMAPI would have reported the outer folder as one broken mod.
func Scan(root string) ([]Mod, error) {
	mods, _, err := ScanInvalid(root)
	return mods, err
}

// ScanInvalid is Scan that also returns the first manifest it skipped for not parsing, nil when there was none.
func ScanInvalid(root string) ([]Mod, *Invalid, error) {
	base, err := safefs.EvalSymlinks(root)
	if err != nil {
		return nil, nil, err
	}
	var mods []Mod
	var invalid *Invalid
	var walk func(dir, rel string) error
	walk = func(dir, rel string) error {
		b, err := ReadFile(dir)
		if err == nil {
			if m, perr := Parse(b); perr == nil {
				mods = append(mods, Mod{Manifest: m, Folder: rel})
			} else if invalid == nil {
				invalid = &Invalid{Folder: rel, Err: perr}
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		items, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, it := range items {
			if strings.HasPrefix(it.Name(), ".") {
				continue
			}
			child := filepath.Join(dir, it.Name())
			if !safefs.RealDirUnder(base, child) {
				continue
			}
			next := it.Name()
			if rel != "." {
				next = rel + "/" + next
			}
			if err := walk(child, next); err != nil {
				return err
			}
		}
		return nil
	}
	err = walk(root, ".")
	return mods, invalid, err
}

// LoaderManaged reports whether a mod is installed and kept current by a loader or Mortar itself, so a profile never
// lists, imports or downloads it as one of its own mods.
func LoaderManaged(id mod.ID) bool {
	switch id.Fold() {
	case "smapi:smapi.consolecommands", "smapi:smapi.savebackup", "smapi:rethunk.mortarsmapibridge", "bepinex:Rethunk.MortarBepInExBridge":
		return true
	}
	return false
}

// ModID is the manifest's mod as a mod.ID.
func (m Manifest) ModID() mod.ID {
	if m.Format != "" {
		return mod.NewID(m.Format, m.UniqueID)
	}
	return mod.SMAPI(m.UniqueID)
}

// ContentPackForID is the ContentPackFor framework as a mod.ID, empty when the manifest declares none.
func (m Manifest) ContentPackForID() mod.ID {
	if m.ContentPackFor == "" {
		return ""
	}
	return mod.SMAPI(m.ContentPackFor)
}

// WithModID returns the manifest naming id, for a manifest built from another source's data.
func (m Manifest) WithModID(id mod.ID) Manifest {
	m.UniqueID, m.Format = id.Local(), ""
	if f := id.Format(); f != mod.FormatSMAPI {
		m.Format = f
	}
	return m
}
