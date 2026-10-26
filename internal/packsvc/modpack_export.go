package packsvc

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// ModpackResult is what ExportModpack wrote. LeftOut names the mods from a source other than Thunderstore, which a
// modpack cannot depend on; Disabled names the Thunderstore packages left out because the profile switches them off.
type ModpackResult struct {
	Path         string   `json:"path"`
	Dependencies []string `json:"dependencies"`
	LeftOut      []string `json:"leftOut"`
	Disabled     []string `json:"disabled"`
	Configs      int      `json:"configs"`
}

// ModpackOptions is the modpack form. An empty Readme is generated from what the profile holds, and an empty IconPath
// writes a plain icon; Configs are the config files (slash paths below BepInEx/config) that go in.
type ModpackOptions struct {
	Name            string   `json:"name"`
	Author          string   `json:"author"`
	Version         string   `json:"version"`
	Description     string   `json:"description"`
	Website         string   `json:"website"`
	IconPath        string   `json:"iconPath"`
	Readme          string   `json:"readme"`
	Changelog       string   `json:"changelog"`
	IncludeDisabled bool     `json:"includeDisabled"`
	Configs         []string `json:"configs"`
}

// ModpackForm is what the form opens with: the remembered fields (or defaults), the profile's config files, what the
// export would hold, and the changelog against the last export.
type ModpackForm struct {
	Options ModpackOptions `json:"options"`
	// Configs are every config file the profile holds.
	Configs      []string `json:"configs"`
	Dependencies int      `json:"dependencies"`
	LeftOut      []string `json:"leftOut"`
	Disabled     []string `json:"disabled"`
	// LastVersion is the version of the last export, empty when there was none.
	LastVersion string `json:"lastVersion"`
}

const (
	maxModpackDescription = 250
	maxIconBytes          = 4 << 20
	// iconSize is the edge of the square icon Thunderstore requires.
	iconSize = 256
)

var (
	// notPackageName is what Thunderstore refuses in a package name.
	notPackageName = regexp.MustCompile(`[^A-Za-z0-9_]+`)
	packageNameOK  = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	versionNumber  = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
)

// nextVersion is the patch bump of a version; the first export is 1.0.0.
func nextVersion(last string) string {
	m := versionNumber.FindStringSubmatch(last)
	if m == nil {
		return "1.0.0"
	}
	patch, _ := strconv.Atoi(m[3])
	return m[1] + "." + m[2] + "." + strconv.Itoa(patch+1)
}

// modpackPackages sorts the profile's entries into what a modpack depends on, what it cannot (other sources) and what
// the profile switches off.
func modpackPackages(p profile.Profile, includeDisabled bool) ModpackResult {
	res := ModpackResult{Dependencies: []string{}, LeftOut: []string{}, Disabled: []string{}}
	for _, e := range p.WithoutDev().Entries {
		switch {
		case e.Source.Bundled():
		case e.Source.Kind != profile.KindThunderstore:
			res.LeftOut = append(res.LeftOut, entryName(e))
		case allOff(e) && !includeDisabled:
			res.Disabled = append(res.Disabled, e.Source.Name)
		default:
			res.Dependencies = append(res.Dependencies, e.Source.Name+"-"+e.Source.Version)
		}
	}
	return res
}

// ModpackForm reads the form's starting point for the profile.
func (s *Service) ModpackForm(gameID, profileID string) (ModpackForm, error) {
	p, err := s.find(gameID, profileID)
	if err != nil {
		return ModpackForm{}, err
	}
	d := profile.ModpackDraft{}
	if p.Modpack != nil {
		d = *p.Modpack
	}
	cfgs, err := s.configFiles(gameID, profileID)
	if err != nil {
		return ModpackForm{}, err
	}
	names := make([]string, len(cfgs))
	for i, f := range cfgs {
		names[i] = strings.TrimPrefix(f.Path, "config/")
	}
	slices.Sort(names)
	o := ModpackOptions{
		Name: cmp.Or(d.Name, packageName(p.Name)), Author: d.Author, Version: nextVersion(d.Version),
		Description: cmp.Or(d.Description, truncate("Modpack exported from the Mortar profile "+p.Name, maxModpackDescription)),
		Website:     d.Website, IconPath: d.Icon, Readme: d.Readme, IncludeDisabled: d.IncludeDisabled, Configs: names,
	}
	res := modpackPackages(p, o.IncludeDisabled)
	o.Changelog = modpackChangelog(o.Version, d.Exported, res.Dependencies, d.Changelog)
	return ModpackForm{Options: o, Configs: names, Dependencies: len(res.Dependencies), LeftOut: res.LeftOut, Disabled: res.Disabled, LastVersion: d.Version}, nil
}

// ModpackChangelog writes the changelog entry for version against the profile's last export, above the changelog the
// last export carried.
func (s *Service) ModpackChangelog(gameID, profileID, version string, includeDisabled bool) (string, error) {
	p, err := s.find(gameID, profileID)
	if err != nil {
		return "", err
	}
	d := profile.ModpackDraft{}
	if p.Modpack != nil {
		d = *p.Modpack
	}
	return modpackChangelog(version, d.Exported, modpackPackages(p, includeDisabled).Dependencies, d.Changelog), nil
}

// splitDependency is the package id and version of a Namespace-Name-version string.
func splitDependency(dep string) (id, version string) {
	id, version, ok := deps.ThunderstoreRef(dep)
	if !ok {
		return dep, ""
	}
	return id, version
}

// modpackChangelog is the "## version" section listing what differs between two exports' packages, then the older
// changelog. With no earlier export there is nothing to compare, so only the older text (none) remains.
func modpackChangelog(version string, before, after []string, older string) string {
	if before == nil {
		return older
	}
	was := map[string]string{}
	for _, d := range before {
		id, v := splitDependency(d)
		was[id] = v
	}
	var lines []string
	for _, d := range after {
		id, v := splitDependency(d)
		switch old, ok := was[id]; {
		case !ok:
			lines = append(lines, fmt.Sprintf("- Added %s %s", id, v))
		case old != v:
			lines = append(lines, fmt.Sprintf("- Updated %s from %s to %s", id, old, v))
		}
		delete(was, id)
	}
	for id := range was {
		lines = append(lines, "- Removed "+id)
	}
	slices.Sort(lines)
	if len(lines) == 0 {
		lines = []string{"- No package changes"}
	}
	section := "## " + version + "\n\n" + strings.Join(lines, "\n") + "\n"
	if older == "" {
		return section
	}
	return section + "\n" + older
}

// ExportModpackDialog asks where to save, then exports as ExportModpack does; an empty Path means the player cancelled.
func (s *Service) ExportModpackDialog(ctx context.Context, gameID, profileID string, o ModpackOptions) (ModpackResult, error) {
	if s.App == nil {
		return ModpackResult{}, errors.New("no window to ask where to save")
	}
	if _, err := s.find(gameID, profileID); err != nil {
		return ModpackResult{}, err
	}
	dest, err := s.App.SaveFile(winhost.Dialog{Filename: packageName(o.Name) + "-" + o.Version + ".zip", Filters: []winhost.Filter{{Name: i18n.T("{source} modpack (zip)", "source", "Thunderstore"), Pattern: "*.zip"}}})
	if err != nil || dest == "" {
		return ModpackResult{Dependencies: []string{}, LeftOut: []string{}, Disabled: []string{}}, err
	}
	return s.ExportModpack(gameID, profileID, dest, o)
}

// check refuses what Thunderstore would refuse on upload, so the player learns before the zip is made.
func (o ModpackOptions) check() error {
	switch {
	case !packageNameOK.MatchString(o.Name):
		return usererr.New(usererr.Invalid, "the name may hold only letters, digits and underscores")
	case !versionNumber.MatchString(o.Version):
		return usererr.New(usererr.Invalid, "the version must look like 1.0.0")
	case len([]rune(o.Description)) > maxModpackDescription:
		return usererr.New(usererr.Invalid, "the description is longer than 250 characters")
	case o.Website != "" && !strings.HasPrefix(o.Website, "https://") && !strings.HasPrefix(o.Website, "http://"):
		return usererr.New(usererr.Invalid, "the website must be an http or https address")
	}
	return nil
}

// ExportModpack writes the profile as a Thunderstore modpack zip at dest: manifest.json depending on each enabled
// Thunderstore package at its installed version (with IncludeDisabled, the switched-off ones too), the icon, README.md,
// a changelog file when there is one, and the chosen files of the profile's BepInEx/config folder as config/, where
// r2modman and Mortar's own import read them. The form is remembered on the profile for the next export.
func (s *Service) ExportModpack(gameID, profileID, dest string, o ModpackOptions) (ModpackResult, error) {
	p, err := s.find(gameID, profileID)
	if err != nil {
		return ModpackResult{}, err
	}
	if err := o.check(); err != nil {
		return ModpackResult{}, err
	}
	res := modpackPackages(p, o.IncludeDisabled)
	res.Path = dest
	if len(res.Dependencies) == 0 {
		return res, errors.New("the profile holds no enabled Thunderstore packages")
	}
	files := map[string][]byte{}
	if len(o.Configs) > 0 {
		cfgs, err := s.configFiles(gameID, profileID)
		if err != nil {
			return res, err
		}
		for _, name := range o.Configs {
			i := slices.IndexFunc(cfgs, func(f pack.File) bool { return f.Path == "config/"+name })
			if i < 0 {
				return res, usererr.New(usererr.Invalid, "the profile has no config file "+name)
			}
			files[cfgs[i].Path] = cfgs[i].Data
		}
		res.Configs = len(o.Configs)
	}
	manifest, err := json.MarshalIndent(map[string]any{
		"name":           o.Name,
		"version_number": o.Version,
		"website_url":    o.Website,
		"description":    o.Description,
		"dependencies":   res.Dependencies,
	}, "", "  ")
	if err != nil {
		return res, err
	}
	files["manifest.json"] = manifest
	files["README.md"] = []byte(o.Readme)
	if o.Readme == "" {
		files["README.md"] = readme(o, res)
	}
	if o.Changelog != "" {
		files["CHANGELOG.md"] = []byte(o.Changelog)
	}
	if files["icon.png"], err = iconFile(o.IconPath); err != nil {
		return res, err
	}
	zipped, err := pack.Zip(pack.SortedFiles(files))
	if err != nil {
		return res, err
	}
	if err := fsx.WriteFile(dest, zipped, 0o600); err != nil {
		return res, err
	}
	_, err = s.Profiles.SetModpackDraft(gameID, profileID, profile.ModpackDraft{
		Name: o.Name, Author: o.Author, Version: o.Version, Description: o.Description, Website: o.Website, Icon: o.IconPath,
		Readme: o.Readme, Changelog: o.Changelog, IncludeDisabled: o.IncludeDisabled, Exported: res.Dependencies,
	})
	return res, err
}

// configFiles are the profile's BepInEx/config files as a pack holds them, under config/.
func (s *Service) configFiles(gameID, profileID string) ([]pack.File, error) {
	dir, err := s.Profiles.ProfileDir(gameID, profileID)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(dir, "BepInEx", "config")
	var out []pack.File
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := fsx.ReadFile(path)
		out = append(out, pack.File{Path: "config/" + filepath.ToSlash(rel), Data: data})
		return err
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return out, nil
}

func (s *Service) find(gameID, profileID string) (profile.Profile, error) {
	all, err := s.Profiles.List(gameID)
	if err != nil {
		return profile.Profile{}, err
	}
	for _, p := range all {
		if p.ID == profileID {
			return p, nil
		}
	}
	return profile.Profile{}, fmt.Errorf("no profile %q", profileID)
}

func entryName(e profile.Entry) string {
	if len(e.Mods) == 1 && e.Mods[0].Name != "" {
		return e.Mods[0].Name
	}
	if e.Source.Name != "" {
		return e.Source.Name
	}
	return e.Key
}

func packageName(n string) string {
	if n = strings.Trim(notPackageName.ReplaceAllString(n, "_"), "_"); n == "" {
		return "Modpack"
	}
	return n
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func readme(o ModpackOptions, res ModpackResult) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", o.Name)
	if o.Author != "" {
		fmt.Fprintf(&b, "By %s.\n\n", o.Author)
	}
	fmt.Fprintf(&b, "A modpack of %d Thunderstore packages, exported from a Mortar profile.\n", len(res.Dependencies))
	if len(res.LeftOut) > 0 {
		b.WriteString("\n## Not included\n\nThese mods come from somewhere other than Thunderstore, so a modpack cannot install them; get them from their own pages:\n\n")
		for _, n := range res.LeftOut {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}
	if len(res.Disabled) > 0 {
		b.WriteString("\n## Switched off\n\nThe profile had these switched off, so they are left out:\n\n")
		for _, n := range res.Disabled {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}
	return []byte(b.String())
}

// iconFile is the PNG at path, which Thunderstore requires to be 256x256; an empty path gives a plain icon.
func iconFile(path string) ([]byte, error) {
	if path == "" {
		return plainIcon()
	}
	if fi, err := os.Stat(path); err != nil {
		return nil, usererr.Wrap(usererr.Invalid, err)
	} else if fi.Size() > maxIconBytes {
		return nil, usererr.New(usererr.Invalid, "the icon is larger than 4 MB")
	}
	data, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, usererr.New(usererr.Invalid, "the icon is not a PNG image")
	}
	if cfg.Width != iconSize || cfg.Height != iconSize {
		return nil, usererr.New(usererr.Invalid, fmt.Sprintf("the icon is %dx%d; Thunderstore needs %dx%d", cfg.Width, cfg.Height, iconSize, iconSize))
	}
	return data, nil
}

// plainIcon is a plain 256x256 PNG.
func plainIcon() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 0x3a, G: 0x4a, B: 0x5c, A: 0xff}}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}
