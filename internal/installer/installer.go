// Package installer lays an extracted archive out as the files a game's content targets receive. Each driver knows one
// archive shape: a FOMOD installer, a Thunderstore package for a BepInEx game, or a plain folder of mods.
package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/fomod"
	"github.com/Rethunk-Tech/mortar/internal/winname"
)

// Archive is an archive already extracted into Dir (the store item folder).
type Archive struct {
	Dir string
	FS  fs.FS
	// Key is the package's identity: the folder name of a plain install and the Namespace-Name of a Thunderstore package.
	Key string
	// Eval is what a FOMOD's conditions read: the files already installed and the game's version.
	Eval fomod.EvalContext
}

// Open makes an Archive of an extracted folder.
func Open(dir, key string) Archive { return Archive{Dir: dir, FS: os.DirFS(dir), Key: key} }

// Choices are a FOMOD installer's answers.
type Choices = fomod.Choices

// Target is a place in the game that receives content: Root names it, and MaxDepth caps, per lower-case file
// extension, how many folders deep a file of that kind may sit below Root (a game that only reads shallow scripts).
type Target struct {
	ID       string
	Root     string
	MaxDepth map[string]int
	// KeepWhole lists the lower-case extensions that make a folder-loader archive one unit (components.TargetDef.KeepWhole).
	KeepWhole []string
	// Extensions are the lower-case extensions of the files a folder-loader archive lays out here, not in mods.
	Extensions []string
	// Shared marks a target whose files are placed at install in a folder every profile uses (components.TargetDef.Role).
	Shared bool
}

// IsShared reports whether target id is placed at install in a folder every profile shares.
func (g Game) IsShared(id string) bool {
	return slices.ContainsFunc(g.Targets, func(t Target) bool { return t.ID == id && t.Shared })
}

// Game is what a driver needs to know of the game: the loaders it runs and the targets it has.
type Game struct {
	Loaders []string
	Targets []Target
}

// Splits reports whether a folder-loader game takes the layout as one entry per file: the game reads the mods target
// as loose files, and none of the layout's files in it is of a kind that only works beside its siblings.
func (g Game) Splits(l Layout) bool {
	if !slices.Contains(g.Loaders, "folder") {
		return false
	}
	ti := slices.IndexFunc(g.Targets, func(t Target) bool { return t.ID == TargetMods })
	if ti < 0 {
		return false
	}
	keep := g.Targets[ti].KeepWhole
	return !slices.ContainsFunc(l.Files, func(f File) bool {
		return f.Target == TargetMods && slices.Contains(keep, strings.ToLower(strings.TrimPrefix(path.Ext(f.Rel), ".")))
	})
}

// Target ids the drivers write to.
const (
	// TargetMods is the profile's mods folder.
	TargetMods = "mods"
	// TargetProfile is the profile's root folder, where BepInEx keeps its plugins and config.
	TargetProfile = "profile"
)

// File is one file of a layout: Src is its slash path in the archive, Rel where it goes below the target's root.
type File struct{ Src, Target, Rel string }

// Layout is where an archive's files go.
type Layout struct{ Files []File }

// Installer is one archive shape.
type Installer interface {
	ID() string
	// Detect reports whether the archive is this shape for game g.
	Detect(a Archive, g Game) bool
	Layout(a Archive, g Game, choices Choices) (Layout, error)
}

// detectionOrder is the order drivers are asked: the most specific shape first, plain last because it takes anything.
var detectionOrder = []string{"fomod", "thunderstore-rules", "plain"}

var (
	mu       sync.RWMutex
	registry = map[string]Installer{}
)

// Register adds a driver and reports true; a driver calls it in a blank package variable.
func Register(i Installer) bool {
	mu.Lock()
	defer mu.Unlock()
	registry[i.ID()] = i
	return true
}

// Get returns the registered driver with this id.
func Get(id string) (Installer, bool) {
	mu.RLock()
	defer mu.RUnlock()
	i, ok := registry[id]
	return i, ok
}

// Pick returns the first driver, in detection order, that takes the archive.
func Pick(a Archive, g Game) (Installer, bool) {
	for _, id := range detectionOrder {
		if i, ok := Get(id); ok && i.Detect(a, g) {
			return i, true
		}
	}
	return nil, false
}

// ErrUnsafe is a layout entry that would write outside its target or that Windows could not store as named.
var ErrUnsafe = errors.New("unsafe path in layout")

// validate checks every file of l against the game's targets: the target exists, the path stays inside it, every
// name is one Windows keeps as written, and the depth limit for its extension holds, counted after the driver's prefix.
func validate(l Layout, g Game) error {
	for _, f := range l.Files {
		ti := slices.IndexFunc(g.Targets, func(t Target) bool { return t.ID == f.Target })
		if ti < 0 {
			return fmt.Errorf("%s: the game has no %q target", f.Src, f.Target)
		}
		if f.Rel == "" || path.IsAbs(f.Rel) || path.Clean(f.Rel) != f.Rel || strings.HasPrefix(f.Rel, "../") {
			return fmt.Errorf("%w: %q", ErrUnsafe, f.Rel)
		}
		segs := strings.Split(f.Rel, "/")
		for _, s := range segs {
			if !winname.Valid(s) {
				return fmt.Errorf("%w: %q", ErrUnsafe, f.Rel)
			}
		}
		ext := strings.ToLower(strings.TrimPrefix(path.Ext(f.Rel), "."))
		if limit, ok := g.Targets[ti].MaxDepth[ext]; ok && len(segs)-1 > limit {
			return fmt.Errorf("%s: .%s files may sit at most %d folders deep in %s", f.Rel, ext, limit, f.Target)
		}
	}
	return nil
}

// skip is the archive junk a store item never keeps: Mac resource folders and .DS_Store, git data, Thumbs.db and the
// store's own completion marker. Other dot names stay, because a dot-named mod folder is a switched-off mod.
func skip(rel string) bool {
	for seg := range strings.SplitSeq(rel, "/") {
		switch {
		case strings.EqualFold(seg, "__MACOSX"), strings.EqualFold(seg, "Thumbs.db"), seg == ".git", seg == ".DS_Store", seg == ".complete":
			return true
		}
	}
	return false
}

// files lists every regular file of the archive under dir (a slash path, "." for all), as paths relative to dir.
func files(a Archive, dir string) ([]string, error) {
	var out []string
	err := fs.WalkDir(a.FS, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel := p
		if dir != "." {
			rel = strings.TrimPrefix(p, dir+"/")
		}
		out = append(out, rel)
		return nil
	})
	return out, err
}
