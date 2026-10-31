package devsvc

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// assetRecordFile marks a project made only of files: .mortar/project.json, with what manifest.json has no field for.
const assetRecordFile = "project.json"

type assetRecord struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Author string `json:"author"`
	Team   string `json:"team"`
	// Csproj is a code project's project file, a slash path from the project's folder; only kindCode records have one.
	Csproj string `json:"csproj,omitempty"`
}

// A record's Kind is a catalog asset kind, or one of these two that every game has.
const (
	// kindCode is a C# project the player already had: built from the record's Csproj, not from the template's layout.
	kindCode = "code"
	// kindGeneric is a folder of files that installs as it is, with no catalog kind behind it.
	kindGeneric = "generic"
)

// readRecord reads the project's .mortar/project.json of any kind; false when it is missing, damaged or names nothing
// valid.
func readRecord(dir string) (assetRecord, bool) {
	body, err := safefs.ReadFile(filepath.Join(dir, ".mortar", assetRecordFile))
	var rec assetRecord
	if err != nil || json.Unmarshal(body, &rec) != nil || rec.Kind == "" || rec.Team != "" && !teamPattern.MatchString(rec.Team) {
		return assetRecord{}, false
	}
	if rec.Kind == kindCode {
		return rec, codeName.MatchString(rec.Name) && validCsproj(rec.Csproj)
	}
	return rec, modName.MatchString(rec.Name)
}

// readAssetRecord is readRecord for a project made only of files.
func readAssetRecord(dir string) (assetRecord, bool) {
	rec, ok := readRecord(dir)
	return rec, ok && rec.Kind != kindCode
}

// validCsproj keeps a record from a cloned repository inside the project's folder.
func validCsproj(p string) bool {
	return fs.ValidPath(p) && !strings.Contains(p, "\\") && strings.HasSuffix(strings.ToLower(p), ".csproj")
}

// AssetKinds are the mods made only of files that the game's catalog entry offers beside code: its asset kinds less
// those the content pack supersedes, then the content pack (whose Recommends are its folders' mods).
func (s *Service) AssetKinds(gameID string) []components.AssetKind {
	tpl, _ := Template(gameID)
	folders := contentFolders(gameID)
	out := []components.AssetKind{}
	for _, k := range tpl.AssetKinds {
		if !superseded(folders, k) {
			out = append(out, k)
		}
	}
	if len(folders) > 0 {
		out = append(out, contentAssetKind(folders))
	}
	return out
}

// assetKind is any kind a project may be recorded as, a superseded one included.
func (s *Service) assetKind(gameID, id string) (components.AssetKind, error) {
	if id == kindGeneric {
		return components.AssetKind{ID: kindGeneric}, nil
	}
	tpl, _ := Template(gameID)
	all := tpl.AssetKinds
	if folders := contentFolders(gameID); len(folders) > 0 {
		all = append(slices.Clone(all), contentAssetKind(folders))
	}
	i := slices.IndexFunc(all, func(k components.AssetKind) bool { return k.ID == id })
	if i < 0 {
		return components.AssetKind{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no %q kind of mod", gameID, id))
	}
	return all[i], nil
}

// createAsset starts a project of the kind m.Kind: a package folder Mortar packs itself, so no SDK is needed. Its
// dependency joins later through AddDependency, as the wizard's choices do for code.
func (s *Service) createAsset(ctx context.Context, m NewMod) (_ settings.DevProject, err error) {
	kind, err := s.assetKind(m.Game, m.Kind)
	if err != nil {
		return settings.DevProject{}, err
	}
	ticked, err := tickedContent(m.Game, m.Values, kind.ID == kindContent)
	if err != nil {
		return settings.DevProject{}, err
	}
	author := cmp.Or(m.Values["modAuthor"], m.Owner)
	if err := (ProjectEdit{Name: m.Name, Description: m.Values["description"], Author: author, Team: m.Values["team"]}).checkDetails(); err != nil {
		return settings.DevProject{}, err
	}
	dest, url, err := m.destination()
	if err != nil {
		return settings.DevProject{}, err
	}
	defer removeIfFailed(dest, &err)
	manifest, err := json.MarshalIndent(thunderstoreManifest{
		Name: m.Name, VersionNumber: "0.1.0", Description: m.Values["description"], WebsiteURL: &url, Dependencies: []string{},
	}, "", "  ")
	if err != nil {
		return settings.DevProject{}, err
	}
	record, err := json.MarshalIndent(assetRecord{Kind: kind.ID, Name: m.Name, Author: author, Team: m.Values["team"]}, "", "  ")
	if err != nil {
		return settings.DevProject{}, err
	}
	files := map[string][]byte{
		"manifest.json":              append(manifest, '\n'),
		"README.md":                  []byte("# " + m.Name + "\n\n" + m.Values["description"] + "\n"),
		"CHANGELOG.md":               []byte("# Changelog\n"),
		"icon.png":                   placeholderIcon(),
		".gitignore":                 []byte("artifacts/\n"),
		"LICENSE":                    []byte(mitLicense(author)),
		".mortar/" + assetRecordFile: append(record, '\n'),
	}
	if kind.ID != kindContent {
		files[kind.Folder+"/README.txt"] = []byte(kind.Label + ": put " + strings.Join(kind.Patterns, ", ") + " files in this folder.\n")
	}
	maps.Copy(files, contentStarters(packContentRoot, ticked))
	for name, body := range kind.Files {
		files[name] = []byte(strings.ReplaceAll(body, "{name}", m.Name))
	}
	if err := writeFiles(dest, files); err != nil {
		return settings.DevProject{}, err
	}
	if err := s.applyLicense(ctx, dest, m.License); err != nil {
		return settings.DevProject{}, err
	}
	s.commit(ctx, dest, false, m.Name)
	if m.GitHub {
		visibility := "--public"
		if m.Private {
			visibility = "--private"
		}
		if _, err := s.run(ctx, dest, "gh", "repo", "create", m.Owner+"/"+m.Repo, visibility, "--source", "."); err != nil {
			return settings.DevProject{}, err
		}
	}
	return s.enroll(m.Game, dest, m.Name, m.Profile, m.CopyFrom, kind.ID, false)
}

func writeFiles(dir string, files map[string][]byte) error {
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			return err
		}
		if err := safefs.WriteFile(p, body, 0o644); err != nil { // #nosec G306 -- a tracked project file
			return err
		}
	}
	return nil
}

// placeholderIcon is a plain 256×256 PNG for a new asset project; Pack says while it is still this one.
func placeholderIcon() []byte {
	img := image.NewRGBA(image.Rect(0, 0, iconSide, iconSide))
	for y := range iconSide {
		for x := range iconSide {
			img.Set(x, y, color.RGBA{R: 0x2b, G: 0x2d, B: 0x31, A: 0xff})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func mitLicense(holder string) string {
	return fmt.Sprintf(`MIT License

Copyright (c) %d %s

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`, time.Now().Year(), holder)
}

// setManifestDependency adds pkg at version to an asset project's manifest.json, replacing any version of it, or
// removes it when version is empty.
func setManifestDependency(dir, pkg, version string) error {
	return editManifest(dir, func(raw map[string]any) {
		deps := []string{}
		if list, ok := raw["dependencies"].([]any); ok {
			for _, d := range list {
				if s, ok := d.(string); ok && !strings.HasPrefix(s, pkg+"-") {
					deps = append(deps, s)
				}
			}
		}
		if version != "" {
			deps = append(deps, pkg+"-"+version)
		}
		raw["dependencies"] = deps
	})
}

// editManifest rewrites manifest.json with edit applied, keeping the fields Mortar does not know.
func editManifest(dir string, edit func(raw map[string]any)) error {
	file := filepath.Join(dir, "manifest.json")
	body, err := safefs.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		// An adopted folder has none until a change needs one.
		body, err = []byte("{}"), nil
	}
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return usererr.Wrap(usererr.Damaged, fmt.Errorf("manifest.json: %w", err))
	}
	edit(raw)
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return safefs.WriteFile(file, append(out, '\n'), 0o644) // #nosec G306 -- a tracked project file
}

// manifestDependencies lists manifest.json's dependencies as packages and versions, the loader left out.
func manifestDependencies(dir string) ([]Dependency, error) {
	body, err := safefs.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m thunderstoreManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, usererr.Wrap(usererr.Damaged, fmt.Errorf("manifest.json: %w", err))
	}
	out := []Dependency{}
	for _, d := range m.Dependencies {
		if i := strings.LastIndex(d, "-"); i > 0 && d[:i] != loaderPackage {
			out = append(out, Dependency{Package: d[:i], Version: d[i+1:]})
		}
	}
	return out, nil
}

// editAsset writes an asset project's description into manifest.json and its author and team into its record; the
// name stays, since the folder, the record and the package are named by it.
func editAsset(p settings.DevProject, e ProjectEdit) error {
	rec, ok := readAssetRecord(p.Dir)
	if !ok {
		return usererr.Wrap(usererr.NotFound, errors.New("the project's record is missing"))
	}
	if e.Name != rec.Name {
		return usererr.Wrap(usererr.Invalid, errors.New("a project of files keeps its name"))
	}
	rec.Author, rec.Team = e.Author, e.Team
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	if err := safefs.WriteFile(filepath.Join(p.Dir, ".mortar", assetRecordFile), append(body, '\n'), 0o644); err != nil { // #nosec G306 -- a tracked project file
		return err
	}
	return editManifest(p.Dir, func(raw map[string]any) { raw["description"] = e.Description })
}

// assetInfo reads an asset project's details from manifest.json and its record.
func assetInfo(dir string, rec assetRecord) ProjectInfo {
	info := ProjectInfo{Name: rec.Name, Author: rec.Author, Team: rec.Team, Kind: rec.Kind, Categories: []string{}}
	if body, err := safefs.ReadFile(filepath.Join(dir, "manifest.json")); err == nil {
		var m thunderstoreManifest
		if json.Unmarshal(body, &m) == nil {
			info.Description, info.Version = m.Description, m.VersionNumber
		}
	}
	if rec.Team != "" {
		info.Package = rec.Team + "-" + rec.Name
	}
	return info
}

// packAsset zips an asset project the way Thunderstore takes it: manifest.json (at the version given), icon, README,
// CHANGELOG, LICENSE and every folder but Mortar's and git's own, into artifacts/thunderstore/release.
func packAsset(p settings.DevProject, team, version string) (string, error) {
	if !packageVersion.MatchString(version) {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a version like 1.2.3; fix manifest.json", version))
	}
	manifest, err := completeManifest(p.Dir, p.Name, version)
	if err != nil {
		return "", err
	}
	out := filepath.Join(p.Dir, "artifacts", "thunderstore", "release")
	if err := os.MkdirAll(out, 0o750); err != nil {
		return "", err
	}
	zipPath := filepath.Join(out, fmt.Sprintf("%s-%s-%s.zip", cmp.Or(team, "Team"), p.Name, version))
	f, err := safefs.Create(zipPath)
	if err != nil {
		return "", err
	}
	zw := zip.NewWriter(f)
	err = errors.Join(addZipFile(zw, "manifest.json", bytes.NewReader(manifest)), addTree(zw, p.Dir))
	return zipPath, errors.Join(err, zw.Close(), f.Close())
}

// completeManifest is dir's manifest.json at version, with the fields Thunderstore insists on filled in when a folder
// adopted with no manifest, or a partial one, lacks them.
func completeManifest(dir, name, version string) ([]byte, error) {
	manifest, err := safefs.ReadFile(filepath.Join(dir, "manifest.json"))
	raw := map[string]any{}
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := mod.UnmarshalManifest(manifest, &raw); err != nil {
			return nil, usererr.Wrap(usererr.Damaged, fmt.Errorf("manifest.json: %w", err))
		}
	}
	// A folder adopted with no manifest, or a partial one, gets only the fields Thunderstore insists on.
	for field, def := range map[string]any{"name": name, "description": "", "website_url": "", "dependencies": []string{}} {
		if _, ok := raw[field]; !ok {
			raw[field] = def
		}
	}
	raw["version_number"] = version
	return json.MarshalIndent(raw, "", "  ")
}

func addZipFile(zw *zip.Writer, name string, r io.Reader) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	return err
}

// addTree adds the project's files to the package; manifest.json is written apart, at the release's version.
func addTree(zw *zip.Writer, dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	skip := []string{".git", ".mortar", "artifacts", ".gitignore", "manifest.json"}
	return fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil || rel == "." {
			return err
		}
		if slices.Contains(skip, strings.Split(rel, "/")[0]) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || isJunk(d.Name()) {
			return nil
		}
		in, err := root.Open(rel)
		if err != nil {
			return err
		}
		defer func() { _ = in.Close() }()
		return addZipFile(zw, path.Clean(rel), in)
	})
}

// isJunk is a file a package never keeps.
func isJunk(name string) bool {
	return strings.EqualFold(name, "Thumbs.db") || name == ".DS_Store"
}

// packAssetProject packs an asset project at its next version (the latest tag, else the manifest's) and checks it.
func (s *Service) packAssetProject(ctx context.Context, gameID string, p settings.DevProject) (PackResult, error) {
	rec, _ := readAssetRecord(p.Dir)
	version := cmp.Or(assetInfo(p.Dir, rec).Version, "0.1.0")
	if v, err := s.NextVersions(ctx, gameID, p.Dir); err == nil && v.Current != "" {
		version = v.Current
	}
	r := PackResult{OK: true, Issues: []string{}}
	zipPath, err := packAsset(p, rec.Team, version)
	if err != nil {
		return PackResult{}, err
	}
	r.Zip = zipPath
	r.Issues = Validate(zipPath)
	r.Issues = append(r.Issues, contentIssues(p.Dir, packContentRoot, contentFolders(gameID), p.Kind == kindContent)...)
	if p.TemplateIcon != "" && zipFileHash(zipPath, "icon.png") == p.TemplateIcon {
		r.Issues = append(r.Issues, "icon.png is still the placeholder; draw the mod its own")
	}
	return r, nil
}

// AssetForm is what the wizard asks for a mod of the asset kind id: the template's questions for its description,
// author and team under the same ids (so remembered answers carry over), and the kind's packages to start with.
func (s *Service) AssetForm(gameID, id string) (Form, error) {
	kind, err := s.assetKind(gameID, id)
	if err != nil {
		return Form{}, err
	}
	params := assetParams()
	if kind.ID == kindContent {
		folders := contentFolders(gameID)
		return Form{Name: kind.Label, Recommends: contentRecommends(folders), Parameters: append(params, contentQuestion(folders, true))}, nil
	}
	recommends := []components.DevRecommend{{Package: kind.Dependency}}
	for _, r := range kind.Recommends {
		recommends = append(recommends, components.DevRecommend{Package: r})
	}
	if folders := contentFolders(gameID); len(folders) > 0 {
		recommends = append(recommends, contentRecommends(folders)...)
		params = append(params, contentQuestion(folders, false))
	}
	return Form{Name: kind.Label, Recommends: recommends, Parameters: params}, nil
}

func assetParams() []Param {
	return []Param{
		{ID: "description", Label: "Description", Description: "What the mod does, in one line.", Kind: "string", Required: true, MaxLength: maxDescription, Own: true},
		{ID: "modAuthor", Label: "Author", Description: "Who holds the copyright.", Kind: "string", Required: true, Remember: true, Own: true},
		{ID: "team", Label: "Thunderstore team", Description: "The team the mod is published under.", Kind: "string", Required: true, Remember: true, Pattern: teamPattern.String(), Own: true},
	}
}

// OpenAssetFolder shows the folder of an asset project that its loader reads, where the player adds files.
func (s *Service) OpenAssetFolder(gameID, dir string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	kind, err := s.assetKind(gameID, p.Kind)
	if err != nil {
		return err
	}
	if kind.ID == kindGeneric {
		return datadir.Open(p.Dir)
	}
	if kind.ID == kindContent {
		if folder := filepath.Join(p.Dir, filepath.FromSlash(packContentRoot)); isDir(folder) {
			return datadir.Open(folder)
		}
		return datadir.Open(p.Dir)
	}
	folder := filepath.Join(p.Dir, filepath.FromSlash(kind.Folder))
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return err
	}
	return datadir.Open(folder)
}

// AssetFolder is a subfolder of an asset project, which installs as its Root whenever the subfolder changes.
type AssetFolder struct{ Game, Dir, Root string }

// AssetSubfolders are every asset project's folders below its root, for the folder watch, which sees only a folder's
// top level: a file dropped into moresuits/ reinstalls the project. Git's, Mortar's and the packs' folders are left out.
func AssetSubfolders(cur settings.Settings, games []string) []AssetFolder {
	var out []AssetFolder
	for _, g := range games {
		for _, p := range cur.GamePrefs(g).DevProjects {
			if p.Kind == "" {
				continue
			}
			_ = filepath.WalkDir(p.Dir, func(dir string, d fs.DirEntry, err error) error {
				if err != nil {
					return fs.SkipDir // an unreadable folder has nothing to watch
				}
				if !d.IsDir() || dir == p.Dir {
					return nil
				}
				if rel, _ := filepath.Rel(p.Dir, dir); slices.Contains([]string{".git", ".mortar", "artifacts"}, rel) {
					return fs.SkipDir
				}
				out = append(out, AssetFolder{Game: g, Dir: dir, Root: p.Dir})
				return nil
			})
		}
	}
	return out
}

// destination is the new project's folder, which must not exist yet, and its GitHub address when it gets one.
func (m NewMod) destination() (dest, url string, err error) {
	dest = filepath.Join(m.Parent, m.Repo)
	if _, err := os.Lstat(dest); err == nil {
		return "", "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%s already exists", dest))
	}
	if m.GitHub {
		url = "https://github.com/" + m.Owner + "/" + m.Repo
	}
	return dest, url, nil
}
