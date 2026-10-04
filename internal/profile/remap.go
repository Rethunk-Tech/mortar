package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// RemapAsk is a dialog so the user can pick which extracted folder is the mod.
type RemapAsk struct {
	Key    string      `json:"key"`
	Source Source      `json:"source"`
	Tree   []RemapNode `json:"tree"`
	// Variants is set when the archive holds several copies of the same mod; the user picks one folder of them.
	Variants []RemapVariant `json:"variants,omitempty"`
	// Overlay is set when the archive is an optional file without a manifest and the user picks where it goes
	// inside its main mod; answer it with InstallOverlay.
	Overlay *OverlayAsk `json:"overlay,omitempty"`
}

// RemapVariant is one folder of an archive that ships the same mod in several variants.
type RemapVariant struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// RemapNode is one file or folder in the extracted archive.
type RemapNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Dir      bool        `json:"dir"`
	Size     int64       `json:"size"`
	Children []RemapNode `json:"children,omitempty"`
}

// NeedRootError means the store item has no SMAPI-reachable manifest, or several variants of one mod, and the user has
// not chosen a folder.
type NeedRootError struct {
	Ask RemapAsk
}

func (e *NeedRootError) Error() string { return "this archive needs a mod folder chosen" }

func (s *Store) remapAsk(game, id, key string) (RemapAsk, bool, error) {
	dir, err := s.items.Path(game, key)
	if err != nil {
		return RemapAsk{}, false, err
	}
	rel, err := s.items.Root(game, key)
	if err != nil {
		return RemapAsk{}, false, err
	}
	if rel != "" {
		return RemapAsk{}, false, nil
	}
	if leftover := contentRootFile(dir); leftover != "" {
		tree, err := remapTree(dir)
		if err != nil {
			return RemapAsk{}, false, err
		}
		return RemapAsk{Key: key, Tree: tree}, true, nil
	}
	found, err := manifest.Scan(dir)
	if err != nil {
		return RemapAsk{}, false, err
	}
	rawXNB, err := hasRawXNB(dir)
	if err != nil {
		return RemapAsk{}, false, err
	}
	if len(found) == 0 && rawXNB {
		return RemapAsk{}, false, &rawXNBError{Key: key}
	}
	if len(found) == 0 {
		if has, err := hasManifestFile(dir); err != nil {
			return RemapAsk{}, false, err
		} else if !has {
			return RemapAsk{}, false, &NoModError{Key: key}
		}
	}
	vars := variants(found)
	if len(found) > 0 && len(vars) == 0 {
		return RemapAsk{}, false, nil
	}
	if len(vars) > 0 {
		if rel := s.priorVariant(game, id, found, vars); rel != "" {
			return RemapAsk{}, false, s.items.SetRoot(game, key, rel)
		}
	}
	tree, err := remapTree(dir)
	if err != nil {
		return RemapAsk{}, false, err
	}
	return RemapAsk{Key: key, Tree: tree, Variants: vars}, true, nil
}

func hasRawXNB(root string) (bool, error) {
	found := false
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if found {
			return fs.SkipAll
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".xnb") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return false, err
	}
	return found, nil
}

// variants returns one folder per copy of each mod the archive holds more than once, since SMAPI refuses to load a
// UniqueID twice. Each folder is widened to its highest ancestor that still holds a single copy, so a variant that
// bundles several mods ("Option A/[CP] Mod", "Option A/[JA] Mod") is picked whole.
func variants(found []manifest.Mod) []RemapVariant {
	byID := map[string][]manifest.Mod{}
	for _, m := range found {
		id := strings.ToLower(m.UniqueID)
		byID[id] = append(byID[id], m)
	}
	under := func(dir, folder string) bool { return folder == dir || strings.HasPrefix(folder, dir+"/") }
	single := func(dir string) bool {
		for _, ms := range byID {
			n := 0
			for _, m := range ms {
				if under(dir, m.Folder) {
					n++
				}
			}
			if n > 1 {
				return false
			}
		}
		return true
	}
	var out []RemapVariant
	for _, m := range found {
		if len(byID[strings.ToLower(m.UniqueID)]) < 2 {
			continue
		}
		dir := m.Folder
		for p := path.Dir(dir); p != "." && single(p); p = path.Dir(p) {
			dir = p
		}
		if !slices.ContainsFunc(out, func(v RemapVariant) bool { return v.Path == dir }) {
			out = append(out, RemapVariant{Path: dir, Version: m.Version, Description: m.Description})
		}
	}
	return out
}

// priorVariant is the variant folder the profile's current entry for these mods was installed from, when this
// archive has a folder of the same name, so an update keeps the variant the user chose.
func (s *Store) priorVariant(game, id string, found []manifest.Mod, vars []RemapVariant) string {
	p, err := s.read(game, id)
	if err != nil {
		return ""
	}
	for _, e := range p.Entries {
		if !slices.ContainsFunc(e.Mods, func(m EntryMod) bool {
			return slices.ContainsFunc(found, func(f manifest.Mod) bool { return SameID(f.UniqueID, m.UniqueID) })
		}) {
			continue
		}
		rel, err := s.items.Root(game, e.Key)
		if err == nil && rel != "" && slices.ContainsFunc(vars, func(v RemapVariant) bool { return v.Path == rel }) {
			return rel
		}
	}
	return ""
}

func contentRootFile(dir string) string {
	b, err := fsx.ReadFile(filepath.Join(dir, store.RootFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func remapTree(root string) ([]RemapNode, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []RemapNode
	for _, e := range ents {
		if e.Name() == store.RootFile || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		n, err := remapNode(root, e.Name(), e)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func remapNode(root, rel string, e fs.DirEntry) (RemapNode, error) {
	p := filepath.Join(root, filepath.FromSlash(rel))
	n := RemapNode{Name: e.Name(), Path: filepath.ToSlash(rel), Dir: e.IsDir()}
	if !e.IsDir() {
		info, err := e.Info()
		if err != nil {
			return RemapNode{}, err
		}
		n.Size = info.Size()
		return n, nil
	}
	children, err := os.ReadDir(p)
	if err != nil {
		return RemapNode{}, err
	}
	for _, c := range children {
		if c.Name() == store.RootFile || strings.HasPrefix(c.Name(), ".") {
			continue
		}
		next := c.Name()
		if rel != "" {
			next = rel + "/" + c.Name()
		}
		ch, err := remapNode(root, next, c)
		if err != nil {
			return RemapNode{}, err
		}
		n.Children = append(n.Children, ch)
		n.Size += ch.Size
	}
	return n, nil
}

// InstallRemap records the chosen folder as this store item's root and adds the item to the profile.
func (s *Store) InstallRemap(game, id, key, root string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	dir, err := s.items.Path(game, key)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	rel := store.CleanRoot(root)
	if rel == "" {
		return InstallResult{}, &InstallError{Msg: "Choose a folder that holds a SMAPI manifest", Err: fmt.Errorf("root %q", root)}
	}
	sub := filepath.Join(dir, filepath.FromSlash(rel))
	found, err := manifest.Scan(sub)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	if len(found) == 0 {
		return InstallResult{}, &InstallError{Msg: "Choose a folder that holds a SMAPI manifest", Err: &NoModError{Key: key}}
	}
	if len(variants(found)) > 0 {
		return InstallResult{}, &InstallError{Msg: "Choose a folder that holds only one variant of the mod", Err: fmt.Errorf("root %q holds variants", root)}
	}
	if err := s.items.SetRoot(game, key, rel); err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installKey(game, id, key, source)
}

func (s *Service) InstallRemap(game, id, key, root string, source Source) (InstallResult, error) {
	return s.store.InstallRemap(game, id, key, root, source)
}
