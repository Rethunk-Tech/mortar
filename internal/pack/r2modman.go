package pack

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"

	"github.com/Rethunk-Tech/go-modmanagers/r2modman"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/bytefmt"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	thunderstore = "thunderstore"
	codePrefix   = r2modman.CodePrefix
	exportFile   = "export.r2x"
	// BaseURL is Thunderstore's site, where profile codes are stored.
	BaseURL = "https://thunderstore.io"
)

func ref(m r2modman.Mod) (Ref, error) {
	if !nativeID.MatchString(m.Name) {
		return Ref{}, fmt.Errorf("%q is not a Namespace-Name package", m.Name)
	}
	return Ref{Source: thunderstore, Native: m.Name, Version: m.Version.String(), Disabled: !m.Enabled}, nil
}

func refs(mods []r2modman.Mod) ([]Ref, error) {
	out := make([]Ref, 0, len(mods))
	for _, m := range mods {
		r, err := ref(m)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
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

// Detect accepts a code, a bare key (also inside Gale's import link), or a .r2z path.
func (Code) Detect(in Input) bool {
	return r2modman.IsCode(in.Text) || r2modman.IsCodeKey(in.Text) || strings.EqualFold(filepath.Ext(in.Path), ".r2z")
}

// Parse reads the code, fetching it first when the input is a key.
func (c Code) Parse(ctx context.Context, in Input) (Draft, error) {
	text := in.Text
	if key, ok := r2modman.CodeKey(text); ok {
		fetched, err := c.fetch(ctx, key)
		if err != nil {
			return Draft{}, err
		}
		text = fetched
	} else if in.Path != "" && strings.TrimSpace(text) == "" {
		b, err := readCapped(in.Path, maxInput)
		if err != nil {
			return Draft{}, err
		}
		return parseR2Zip(b)
	}
	if int64(len(text)) > maxInput {
		return Draft{}, errors.New("profile code is too large")
	}
	b, err := r2modman.DecodeCode(text)
	if err != nil {
		return Draft{}, err
	}
	return parseR2Zip(b)
}

func (c Code) fetch(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	client := r2modman.Client{HTTP: c.HTTP, BaseURL: cmp.Or(c.URL, BaseURL), UserAgent: source.UserAgentWithSite("")}
	text, err := client.FetchCode(ctx, key)
	if errors.Is(err, httpkit.ErrNotFound) {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("thunderstore has no profile code %s", key))
	}
	if err != nil {
		return "", fmt.Errorf("thunderstore profile code %s: %w", key, err)
	}
	return text, nil
}

func parseR2Zip(data []byte) (Draft, error) {
	exp, err := r2modman.ReadExport(data)
	if err != nil {
		return Draft{}, err
	}
	packages, err := refs(exp.Mods)
	if err != nil {
		return Draft{}, err
	}
	d := Draft{Name: exp.ProfileName, Game: exp.Community, Packages: packages, Configs: filesUnder(exp.Files, "config/")}
	for _, f := range d.Configs {
		delete(exp.Files, f.Path)
	}
	d.Local, d.Loose = carriedPlugins(filesUnder(exp.Files, ""), d.Packages)
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
func (Profile) Detect(in Input) bool { return in.Path != "" && r2modman.IsProfileFolder(in.Path) }

// Parse reads mods.yml and the profile's BepInEx/config files. Game is the catalog id when GameByFolder knows
// r2modman's folder name for the game (for example "LethalCompany"), else that folder name.
func (p Profile) Parse(_ context.Context, in Input) (Draft, error) {
	prof, err := r2modman.ReadProfile(in.Path)
	if err != nil {
		return Draft{}, err
	}
	packages, err := refs(prof.Mods)
	if err != nil {
		return Draft{}, err
	}
	d := Draft{Name: prof.Name, Game: prof.GameFolder, Packages: packages}
	if id, ok := p.lookup(d.Game); ok {
		d.Game = id
	}
	if d.Configs, err = configFiles(in.Path); err != nil {
		return Draft{}, err
	}
	return d, nil
}

// configFiles reads the files below a profile folder's BepInEx/config, subfolders included.
func configFiles(profileDir string) ([]File, error) {
	found, err := r2modman.Configs(profileDir)
	if err != nil {
		return nil, err
	}
	out := make([]File, len(found))
	for i, f := range found {
		out[i] = File{Path: f.Path, Data: f.Data}
	}
	return out, nil
}

// DataDirs lists the r2modman data folders that exist on this machine: extra (the player's own choice, empty for none),
// then r2modmanPlus-local and the Thunderstore Mod Manager's DataFolder under the user config directory.
func DataDirs(extra string) []string {
	var out []string
	if extra != "" && safefs.IsDir(extra) {
		out = append(out, extra)
	}
	for _, d := range r2modman.DataDirs(r2modman.Bases()...) {
		if !slices.Contains(out, d) {
			out = append(out, d)
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

// readCapped reads a file of at most limit bytes.
func readCapped(file string, limit int64) ([]byte, error) {
	st, err := safefs.Stat(file)
	if err != nil {
		return nil, err
	}
	if st.Size() > limit {
		return nil, fmt.Errorf("%s is larger than %s", filepath.Base(file), bytefmt.Format(limit))
	}
	return safefs.ReadFile(file)
}
