package packsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
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

// notPackageName is what Thunderstore refuses in a package name.
var notPackageName = regexp.MustCompile(`[^A-Za-z0-9_]+`)

// ExportModpackDialog asks where to save, then exports as ExportModpack does; an empty Path means the player cancelled.
func (s *Service) ExportModpackDialog(ctx context.Context, gameID, profileID string, configs bool) (ModpackResult, error) {
	if s.App == nil {
		return ModpackResult{}, errors.New("no window to ask where to save")
	}
	p, err := s.find(gameID, profileID)
	if err != nil {
		return ModpackResult{}, err
	}
	d := s.App.Dialog.SaveFile().SetFilename(packageName(p.Name)+".zip").AddFilter("Thunderstore modpack (zip)", "*.zip")
	d.AttachToWindow(s.App.Window.Current())
	dest, err := d.PromptForSingleSelection()
	if err != nil || dest == "" {
		return ModpackResult{Dependencies: []string{}, LeftOut: []string{}, Disabled: []string{}}, err
	}
	return s.ExportModpack(gameID, profileID, dest, configs)
}

// ExportModpack writes the profile as a Thunderstore modpack zip at dest: manifest.json depending on each enabled
// Thunderstore package at its installed version, icon.png, README.md naming what was left out, and with configs the
// profile's BepInEx/config folder as config/, where r2modman and Mortar's own import read it.
func (s *Service) ExportModpack(gameID, profileID, dest string, configs bool) (ModpackResult, error) {
	p, err := s.find(gameID, profileID)
	if err != nil {
		return ModpackResult{}, err
	}
	res := ModpackResult{Path: dest, Dependencies: []string{}, LeftOut: []string{}, Disabled: []string{}}
	for _, e := range p.Entries {
		switch {
		case e.Source.Kind == profile.SourceSMAPI || e.Source.Kind == profile.SourceMortar:
		case e.Source.Kind != profile.KindThunderstore:
			res.LeftOut = append(res.LeftOut, entryName(e))
		case allOff(e):
			res.Disabled = append(res.Disabled, e.Source.Name)
		default:
			res.Dependencies = append(res.Dependencies, e.Source.Name+"-"+e.Source.Version)
		}
	}
	if len(res.Dependencies) == 0 {
		return res, errors.New("the profile holds no enabled Thunderstore packages")
	}
	files := map[string][]byte{}
	if configs {
		dir, err := s.Profiles.ProfileDir(gameID, profileID)
		if err != nil {
			return res, err
		}
		root := filepath.Join(dir, "BepInEx", "config")
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			data, err := fsx.ReadFile(path)
			files["config/"+filepath.ToSlash(rel)] = data
			return err
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return res, err
		}
		res.Configs = len(files)
	}
	manifest, err := json.MarshalIndent(map[string]any{
		"name":           packageName(p.Name),
		"version_number": "1.0.0",
		"website_url":    "",
		"description":    truncate("Modpack exported from the Mortar profile "+p.Name, 250),
		"dependencies":   res.Dependencies,
	}, "", "  ")
	if err != nil {
		return res, err
	}
	files["manifest.json"] = manifest
	files["README.md"] = readme(p.Name, res)
	if files["icon.png"], err = icon(); err != nil {
		return res, err
	}
	return res, writeZip(dest, files)
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

func readme(name string, res ModpackResult) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nA modpack of %d Thunderstore packages, exported from a Mortar profile.\n", name, len(res.Dependencies))
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

// icon is a plain 256x256 PNG, the size Thunderstore requires.
func icon() ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 0x3a, G: 0x4a, B: 0x5c, A: 0xff}}, image.Point{}, draw.Src)
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}

func writeZip(dest string, files map[string][]byte) error {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return fsx.WriteFile(dest, buf.Bytes(), 0o600)
}
