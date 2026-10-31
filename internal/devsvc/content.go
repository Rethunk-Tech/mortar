package devsvc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	// kindContent is the content pack: a project of files carrying any of the game's content folders.
	kindContent = "content"
	// contentParam is the wizard's question for the content folders a new project ticks, joined with "|".
	contentParam = "mortarContent"
	// packContentRoot holds a content pack's folders: below plugins/, since BepInEx and other managers flatten any
	// other folder at a package's top, so it installs as BepInEx/plugins/<package>/mortar/. A code project keeps them
	// in codeContentRoot, which its build ships under plugins/.
	packContentRoot = "plugins/mortar"
	codeContentRoot = "content/mortar"
)

// contentFolders are the content folders the game's template offers; nil when it has no content pack.
func contentFolders(gameID string) []components.ContentFolder {
	tpl, ok := Template(gameID)
	if !ok || tpl.Content == nil {
		return nil
	}
	return tpl.Content.Folders
}

// superseded is an asset kind whose folder is a content folder's old layout: the content pack replaces it.
func superseded(folders []components.ContentFolder, k components.AssetKind) bool {
	return slices.ContainsFunc(folders, func(f components.ContentFolder) bool { return slices.Contains(f.Legacy, k.Folder) })
}

// migrateKind is the kind a project recorded as kind reads as now: a superseded asset kind is the content pack.
func migrateKind(gameID, kind string) string {
	tpl, ok := Template(gameID)
	if !ok || tpl.Content == nil {
		return kind
	}
	i := slices.IndexFunc(tpl.AssetKinds, func(k components.AssetKind) bool { return k.ID == kind })
	if i >= 0 && superseded(tpl.Content.Folders, tpl.AssetKinds[i]) {
		return kindContent
	}
	return kind
}

func contentAssetKind(folders []components.ContentFolder) components.AssetKind {
	k := components.AssetKind{ID: kindContent, Label: "Content pack", Folder: packContentRoot}
	for _, f := range folders {
		if !slices.Contains(k.Recommends, f.Dependency) {
			k.Recommends = append(k.Recommends, f.Dependency)
		}
	}
	return k
}

// contentQuestion asks which content folders a new project carries; a content pack needs at least one.
func contentQuestion(folders []components.ContentFolder, required bool) Param {
	p := Param{
		ID: contentParam, Label: "Content", Description: "Folders of files the package carries; each adds the mod that loads it.",
		Kind: "choice", Multiple: true, Required: required, Own: true, Choices: []Choice{},
	}
	for _, f := range folders {
		p.Choices = append(p.Choices, Choice{Value: f.ID, Label: f.Label})
	}
	return p
}

// contentRecommends are each content folder's mod, recommended when that folder is ticked.
func contentRecommends(folders []components.ContentFolder) []components.DevRecommend {
	var out []components.DevRecommend
	for _, f := range folders {
		i := slices.IndexFunc(out, func(r components.DevRecommend) bool { return r.Package == f.Dependency })
		if i < 0 {
			out = append(out, components.DevRecommend{Package: f.Dependency, Param: contentParam})
			i = len(out) - 1
		}
		out[i].Values = append(out[i].Values, f.ID)
	}
	return out
}

// tickedContent is the content folders values ticks, each one the game offers.
func tickedContent(gameID string, values map[string]string, required bool) ([]components.ContentFolder, error) {
	folders := contentFolders(gameID)
	var out []components.ContentFolder
	for id := range strings.SplitSeq(values[contentParam], "|") {
		if id == "" {
			continue
		}
		i := slices.IndexFunc(folders, func(f components.ContentFolder) bool { return f.ID == id })
		if i < 0 {
			return nil, usererr.Wrap(usererr.Invalid, fmt.Errorf("%s has no %q content", gameID, id))
		}
		out = append(out, folders[i])
	}
	if required && len(out) == 0 {
		return nil, usererr.Wrap(usererr.Invalid, errors.New("tick at least one kind of content"))
	}
	return out, nil
}

// contentStarters are a README in each ticked folder below root, saying what goes there.
func contentStarters(root string, ticked []components.ContentFolder) map[string][]byte {
	out := map[string][]byte{}
	for _, f := range ticked {
		out[root+"/"+f.ID+"/README.txt"] = []byte(f.Label + ": put " + strings.Join(f.Patterns, ", ") + " files in this folder.\n")
	}
	return out
}

// oldFolders are where a pack may hold f's files outside plugins/mortar/: its mod's old layouts, and mortar/ at the
// package's top, which installs flattened.
func oldFolders(f components.ContentFolder) []string {
	return append(slices.Clone(f.Legacy), "mortar/"+f.ID)
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// detectContent lists the content folders dir carries in the mortar/ layout (a pack's plugins/mortar/, or a code
// project's content/mortar/), and those it carries only in an old or misplaced one.
func detectContent(dir string, folders []components.ContentFolder) (found, legacy []string) {
	found, legacy = []string{}, []string{}
	for _, f := range folders {
		if isDir(filepath.Join(dir, filepath.FromSlash(packContentRoot), f.ID)) || isDir(filepath.Join(dir, filepath.FromSlash(codeContentRoot), f.ID)) {
			found = append(found, f.ID)
		}
		if slices.ContainsFunc(oldFolders(f), func(l string) bool { return isDir(filepath.Join(dir, filepath.FromSlash(l))) }) {
			legacy = append(legacy, f.ID)
		}
	}
	return found, legacy
}

// holdsMatch is whether folder, or a folder below it, holds a file one of patterns takes.
func holdsMatch(folder string, patterns []string) bool {
	hit := false
	_ = filepath.WalkDir(folder, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if !d.IsDir() && slices.ContainsFunc(patterns, func(pat string) bool {
			ok, _ := path.Match(strings.ToLower(pat), strings.ToLower(d.Name()))
			return ok
		}) {
			hit = true
			return fs.SkipAll
		}
		return nil
	})
	return hit
}

// contentIssues are what a release would ship empty: each content folder below root (and, for a pack, each old-layout
// one) that holds no file its mod loads, or a pack with no content folder at all.
func contentIssues(dir, root string, folders []components.ContentFolder, pack bool) []string {
	issues := []string{}
	present := false
	for _, f := range folders {
		dirs := []string{root + "/" + f.ID}
		if pack {
			dirs = append(dirs, f.Legacy...)
			if top := "mortar/" + f.ID; isDir(filepath.Join(dir, filepath.FromSlash(top))) {
				issues = append(issues, top+"/ installs flattened from the package's top; move it under "+root+"/")
			}
		}
		for _, d := range dirs {
			full := filepath.Join(dir, filepath.FromSlash(d))
			if !isDir(full) {
				continue
			}
			present = true
			if !holdsMatch(full, f.Patterns) {
				issues = append(issues, fmt.Sprintf("%s/ holds no %s file", d, strings.Join(f.Patterns, ", ")))
			}
		}
	}
	if pack && !present {
		issues = append(issues, "the content pack has no content folder; add one under "+root+"/")
	}
	return issues
}

// MoveLegacyContent moves the files of dir's old-layout content folders into plugins/mortar/<kind>/, which the player confirms
// when adopting; a file already in the new folder stays where it is. It returns how many files moved.
func (s *Service) MoveLegacyContent(gameID, dir string) (int, error) {
	if !filepath.IsAbs(dir) {
		return 0, usererr.Wrap(usererr.Invalid, errors.New("pick the project's folder"))
	}
	root, err := os.OpenRoot(filepath.Clean(dir))
	if err != nil {
		return 0, err
	}
	defer func() { _ = root.Close() }()
	moved := 0
	for _, f := range contentFolders(gameID) {
		dest := packContentRoot + "/" + f.ID
		for _, l := range oldFolders(f) {
			entries, err := fs.ReadDir(root.FS(), l)
			if err != nil {
				continue
			}
			if err := root.MkdirAll(dest, 0o750); err != nil {
				return moved, err
			}
			for _, e := range entries {
				if !e.Type().IsRegular() {
					continue
				}
				if _, err := root.Lstat(dest + "/" + e.Name()); err == nil {
					continue
				}
				if err := root.Rename(l+"/"+e.Name(), dest+"/"+e.Name()); err != nil {
					return moved, err
				}
				moved++
			}
			// Emptied old folders go, up to the package root; one still holding anything stays.
			for p := l; p != "."; p = path.Dir(p) {
				if root.Remove(p) != nil {
					break
				}
			}
		}
	}
	return moved, nil
}
