package devsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Adoption is what Mortar finds in a folder the player brings into the Develop tab, and what they confirm: the name
// and the kind of project it becomes. Kind is "code", "generic" or one of the game's asset kinds.
type Adoption struct {
	// Known is set for a folder that is already a Mortar project, which Add takes as it is.
	Known bool   `json:"known"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	// Csproj is the project file a code project builds, a slash path from the folder.
	Csproj string `json:"csproj"`
	// Kinds are the kinds the folder can be adopted as.
	Kinds []string `json:"kinds"`
}

// junkDirs are folders a search for the project file does not enter.
var junkDirs = []string{".git", ".mortar", "bin", "obj", "node_modules", "artifacts", ".vs"}

const maxCsprojDepth = 4

// findCsproj picks the project file of a folder of code: the one named for the folder, else the shallowest, else the
// first by name. The path is a slash path from dir.
func findCsproj(dir string) string {
	var found []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		rel, _ := filepath.Rel(dir, p)
		switch {
		case d.IsDir() && (slices.Contains(junkDirs, d.Name()) || strings.Count(filepath.ToSlash(rel), "/") >= maxCsprojDepth):
			return fs.SkipDir
		case !d.IsDir() && d.Type().IsRegular() && strings.EqualFold(filepath.Ext(p), ".csproj"):
			found = append(found, filepath.ToSlash(rel))
		}
		return nil
	})
	folder := strings.ToLower(filepath.Base(dir))
	depth := func(p string) int { return strings.Count(p, "/") }
	named := func(p string) bool { return strings.ToLower(strings.TrimSuffix(path.Base(p), path.Ext(p))) == folder }
	slices.SortFunc(found, func(a, b string) int {
		switch {
		case named(a) != named(b):
			if named(a) {
				return -1
			}
			return 1
		case depth(a) != depth(b):
			return depth(a) - depth(b)
		}
		return strings.Compare(a, b)
	})
	if len(found) == 0 {
		return ""
	}
	return found[0]
}

// findAssetKind is the game's asset kind whose folder in dir holds a file its patterns take.
func (s *Service) findAssetKind(gameID, dir string) string {
	for _, k := range s.AssetKinds(gameID) {
		folder := filepath.Join(dir, filepath.FromSlash(k.Folder))
		hit := false
		_ = filepath.WalkDir(folder, func(_ string, d fs.DirEntry, err error) error {
			if err != nil || hit {
				return fs.SkipDir
			}
			if !d.IsDir() && slices.ContainsFunc(k.Patterns, func(pat string) bool {
				ok, _ := path.Match(strings.ToLower(pat), strings.ToLower(d.Name()))
				return ok
			}) {
				hit = true
				return fs.SkipAll
			}
			return nil
		})
		if hit {
			return k.ID
		}
	}
	return ""
}

// folderName is the name a folder suggests for a mod: its letters and digits, starting with a capital.
func folderName(dir string) string {
	var b strings.Builder
	for _, r := range filepath.Base(dir) {
		if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			if b.Len() == 0 {
				r = []rune(strings.ToUpper(string(r)))[0]
			}
			b.WriteRune(r)
		}
	}
	if b.Len() < 2 {
		return "Mod"
	}
	return b.String()[:min(b.Len(), 64)]
}

// Detect reads what dir would become as a Mortar project, writing nothing. A known project reports itself.
func (s *Service) Detect(gameID, dir string) (Adoption, error) {
	if !filepath.IsAbs(dir) {
		return Adoption{}, usererr.Wrap(usererr.Invalid, errors.New("pick the project's folder"))
	}
	dir = filepath.Clean(dir)
	if rec, ok := readRecord(dir); ok {
		return Adoption{Known: true, Name: rec.Name, Kind: rec.Kind, Csproj: rec.Csproj, Kinds: []string{rec.Kind}}, nil
	}
	if name, ok := projectName(dir); ok {
		return Adoption{Known: true, Name: name, Kind: kindCode, Kinds: []string{kindCode}}, nil
	}
	if !installer.HasFiles(installer.Open(dir, "")) {
		return Adoption{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%s holds no files, so there is nothing to develop", dir))
	}
	a := Adoption{Name: folderName(dir), Kind: kindGeneric, Kinds: []string{}}
	if a.Csproj = findCsproj(dir); a.Csproj != "" {
		a.Kind, a.Name = kindCode, strings.TrimSuffix(path.Base(a.Csproj), path.Ext(a.Csproj))
		a.Kinds = append(a.Kinds, kindCode)
	} else if id := s.findAssetKind(gameID, dir); id != "" {
		a.Kind = id
	}
	for _, k := range s.AssetKinds(gameID) {
		a.Kinds = append(a.Kinds, k.ID)
	}
	a.Kinds = append(a.Kinds, kindGeneric)
	return a, nil
}

// Adopt makes the folder in dir a Mortar project of the confirmed kind and name, as Add does for one that already is.
// The only file it writes into the folder is .mortar/project.json.
func (s *Service) Adopt(ctx context.Context, gameID, dir, profileID string, choice Adoption) (settings.DevProject, error) {
	bad := func(what string, args ...any) (settings.DevProject, error) {
		return settings.DevProject{}, usererr.Wrap(usererr.Invalid, fmt.Errorf(what, args...))
	}
	found, err := s.Detect(gameID, dir)
	if err != nil {
		return settings.DevProject{}, err
	}
	dir = filepath.Clean(dir)
	if found.Known {
		return s.Add(ctx, gameID, dir, profileID)
	}
	if !slices.Contains(found.Kinds, choice.Kind) {
		return bad("%s cannot be adopted as %q", dir, choice.Kind)
	}
	rec := assetRecord{Kind: choice.Kind, Name: choice.Name}
	switch {
	case rec.Kind == kindCode && !codeName.MatchString(rec.Name):
		return bad("the name holds letters, digits, dots, dashes and underscores")
	case rec.Kind != kindCode && !modName.MatchString(rec.Name):
		return bad("the mod name must start with a capital letter and hold only letters and digits")
	}
	if rec.Kind == kindCode {
		rec.Csproj = found.Csproj
	}
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return settings.DevProject{}, err
	}
	file := filepath.Join(dir, ".mortar", assetRecordFile)
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		return settings.DevProject{}, err
	}
	if err := safefs.WriteFile(file, append(body, '\n'), 0o644); err != nil { // #nosec G306 -- a tracked project file
		return settings.DevProject{}, err
	}
	p, err := s.Add(ctx, gameID, dir, profileID)
	if err != nil {
		// A refused adoption leaves the folder as it was found.
		_ = safefs.Remove(file)
		_ = os.Remove(filepath.Dir(file))
	}
	return p, err
}
