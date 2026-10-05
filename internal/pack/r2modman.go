package pack

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"gopkg.in/yaml.v3"
)

const (
	thunderstore = "thunderstore"
	codePrefix   = "#r2modman"
	exportFile   = "export.r2x"
	// BaseURL is Thunderstore's site, where profile codes are stored.
	BaseURL = "https://thunderstore.io"
)

var codeKey = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// r2Mod is one mod in export.r2x (name, version {major,minor,patch}, enabled) and in an r2modman mods.yml, which
// spells the version versionNumber.
type r2Version struct {
	Major int `yaml:"major"`
	Minor int `yaml:"minor"`
	Patch int `yaml:"patch"`
}

type r2Mod struct {
	Name          string    `yaml:"name"`
	Version       r2Version `yaml:"version"`
	VersionNumber r2Version `yaml:"versionNumber"`
	Enabled       *bool     `yaml:"enabled"`
}

func (m r2Mod) ref() Ref {
	v := m.Version
	if v == (r2Version{}) {
		v = m.VersionNumber
	}
	return Ref{
		Source: thunderstore, Native: m.Name, Version: fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch),
		Disabled: m.Enabled != nil && !*m.Enabled,
	}
}

func refs(mods []r2Mod) []Ref {
	out := make([]Ref, 0, len(mods))
	for _, m := range mods {
		out = append(out, m.ref())
	}
	return out
}

// Code reads r2modman and Gale share codes: "#r2modman\n" and the base64 of a zip holding export.r2x and the
// profile's config files. A bare code key, or a .r2z file, is read too. Gale keeps its profiles in a database, so
// its exported codes and .r2z files are how its profiles are read.
type Code struct {
	HTTP *http.Client
	// URL is Thunderstore's address; empty means the real site.
	URL string
}

// ID names the format.
func (Code) ID() string { return "r2modman-code" }

// Detect accepts a code, a bare key, or a .r2z path.
func (Code) Detect(in Input) bool {
	text := strings.TrimSpace(in.Text)
	return strings.HasPrefix(text, codePrefix) || codeKey.MatchString(text) || strings.EqualFold(filepath.Ext(in.Path), ".r2z")
}

// Parse reads the code, fetching it first when the input is a key.
func (c Code) Parse(ctx context.Context, in Input) (Draft, error) {
	var zipBytes []byte
	text := strings.TrimSpace(in.Text)
	switch {
	case in.Path != "" && text == "":
		b, err := fsx.ReadFile(in.Path)
		if err != nil {
			return Draft{}, err
		}
		zipBytes = b
	default:
		if codeKey.MatchString(text) {
			fetched, err := c.fetch(ctx, text)
			if err != nil {
				return Draft{}, err
			}
			text = fetched
		}
		body, ok := strings.CutPrefix(text, codePrefix)
		if !ok {
			return Draft{}, errors.New("not an r2modman profile code")
		}
		b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(body), ""))
		if err != nil {
			return Draft{}, fmt.Errorf("profile code is not base64: %w", err)
		}
		zipBytes = b
	}
	return parseR2Zip(zipBytes)
}

func (c Code) fetch(ctx context.Context, key string) (string, error) {
	base := c.URL
	if base == "" {
		base = BaseURL
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/experimental/legacyprofile/get/"+key+"/", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mortar (+https://mortar.rethunk.tech)")
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thunderstore answered %s for profile code %s", resp.Status, key)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	return string(b), err
}

func parseR2Zip(data []byte) (Draft, error) {
	files, err := readZip(data)
	if err != nil {
		return Draft{}, err
	}
	raw, ok := files[exportFile]
	if !ok {
		return Draft{}, errors.New("profile has no " + exportFile)
	}
	var doc struct {
		Name      string  `yaml:"profileName"`
		Community string  `yaml:"community"`
		Mods      []r2Mod `yaml:"mods"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return Draft{}, fmt.Errorf("reading %s: %w", exportFile, err)
	}
	d := Draft{Name: doc.Name, Game: doc.Community, Packages: refs(doc.Mods), Configs: filesUnder(files, "config/")}
	delete(files, exportFile)
	for _, f := range d.Configs {
		delete(files, f.Path)
	}
	d.Loose = filesUnder(files, "")
	return d, nil
}

// Profile reads an r2modman profile folder, <data>/<Game>/profiles/<name>, which holds mods.yml. It reads only.
type Profile struct {
	// GameByFolder maps r2modman's folder name for a game to the catalog game id (game.ByR2modmanFolder); nil or a
	// miss leaves the folder name in Draft.Game.
	GameByFolder func(folder string) (string, bool)
}

// ID names the format.
func (Profile) ID() string { return "r2modman-profile" }

// Detect accepts a folder holding mods.yml.
func (Profile) Detect(in Input) bool {
	st, err := os.Stat(filepath.Join(in.Path, "mods.yml"))
	return in.Path != "" && err == nil && !st.IsDir()
}

// Parse reads mods.yml and the profile's BepInEx/config files. Game is the catalog id when GameByFolder knows
// r2modman's folder name for the game (for example "LethalCompany"), else that folder name.
func (p Profile) Parse(_ context.Context, in Input) (Draft, error) {
	raw, err := fsx.ReadFile(filepath.Join(in.Path, "mods.yml"))
	if err != nil {
		return Draft{}, err
	}
	var mods []r2Mod
	if err := yaml.Unmarshal(raw, &mods); err != nil {
		return Draft{}, fmt.Errorf("reading mods.yml: %w", err)
	}
	d := Draft{Name: filepath.Base(in.Path), Game: filepath.Base(filepath.Dir(filepath.Dir(in.Path))), Packages: refs(mods)}
	if id, ok := p.lookup(d.Game); ok {
		d.Game = id
	}
	cfg := filepath.Join(in.Path, "BepInEx", "config")
	entries, _ := os.ReadDir(cfg)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := fsx.ReadFile(filepath.Join(cfg, e.Name()))
		if err != nil {
			return Draft{}, err
		}
		d.Configs = append(d.Configs, File{Path: "config/" + e.Name(), Data: b})
	}
	return d, nil
}

// DataDirs lists the r2modman data folders that exist on this machine: r2modmanPlus-local, and the Thunderstore Mod
// Manager's DataFolder. Both live under the user config directory (~/.config on Linux, %AppData% on Windows).
func DataDirs() []string {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range []string{
		filepath.Join(base, "r2modmanPlus-local"),
		filepath.Join(base, "Thunderstore Mod Manager", "DataFolder"),
	} {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

func (p Profile) lookup(folder string) (string, bool) {
	if p.GameByFolder == nil {
		return "", false
	}
	return p.GameByFolder(folder)
}
