package pack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// galeAppID is the folder Gale keeps its data in below the user's data directory
// (Kesomannen/gale src-tauri/src/util/path.rs, APP_GUID and default_app_data_dir).
const galeAppID = "com.kesomannen.gale"

const galeManifest = "profile.json"

// galeMod is one entry of a Gale profile's mods array (src-tauri/src/profile/mod.rs, ProfileMod): a Thunderstore mod
// names its package as fullName, "Namespace-Name-Version"; a mod added from disk has no fullName.
type galeMod struct {
	FullName string `json:"fullName"`
	Enabled  *bool  `json:"enabled"`
}

// Gale reads a Gale profile folder, <data>/<game slug>/profiles/<name>, which holds profile.json (the format Gale
// kept its profiles in before moving them into data.sqlite3, src-tauri/src/db/migrate.rs, legacy::ProfileSaveData).
// It reads only.
type Gale struct {
	// GameBySlug maps Gale's game slug, which is the game's Thunderstore community key, to the catalog game id; nil or
	// a miss leaves the slug in Draft.Game.
	GameBySlug func(slug string) (string, bool)
}

// ID names the format.
func (Gale) ID() string { return "gale-profile" }

// Detect accepts a folder holding profile.json.
func (Gale) Detect(in Input) bool { return in.Path != "" && IsGaleProfileFolder(in.Path) }

// IsGaleProfileFolder reports whether dir is a Gale profile folder: it holds a profile.json.
func IsGaleProfileFolder(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, galeManifest))
	return err == nil && !st.IsDir()
}

// Parse reads profile.json and the profile's BepInEx/config files. Mods Gale added from disk carry no package name
// and are left out.
func (g Gale) Parse(_ context.Context, in Input) (Draft, error) {
	raw, err := readCapped(filepath.Join(in.Path, galeManifest), maxFile)
	if err != nil {
		return Draft{}, err
	}
	var doc struct {
		Mods []galeMod `json:"mods"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Draft{}, fmt.Errorf("reading %s: %w", galeManifest, err)
	}
	d := Draft{Name: filepath.Base(in.Path), Game: filepath.Base(filepath.Dir(filepath.Dir(in.Path)))}
	if g.GameBySlug != nil {
		if id, ok := g.GameBySlug(d.Game); ok {
			d.Game = id
		}
	}
	for _, m := range doc.Mods {
		owner, rest, ok := strings.Cut(m.FullName, "-")
		name, version, ok2 := strings.Cut(rest, "-")
		if !ok || !ok2 || !nativeID.MatchString(owner+"-"+name) {
			continue
		}
		d.Packages = append(d.Packages, Ref{Source: thunderstore, Native: owner + "-" + name, Version: version, Disabled: m.Enabled != nil && !*m.Enabled})
	}
	if d.Configs, err = bepinexConfigs(in.Path); err != nil {
		return Draft{}, err
	}
	return d, nil
}

// GaleDataDirs lists Gale's data folder when it exists on this machine: <user data dir>/com.kesomannen.gale, with the
// user data dir as the dirs crate resolves it (XDG data home on Linux, %AppData% on Windows, Application Support on
// macOS).
func GaleDataDirs() []string {
	var base string
	if runtime.GOOS == "linux" {
		if base = os.Getenv("XDG_DATA_HOME"); base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil
			}
			base = filepath.Join(home, ".local", "share")
		}
	} else {
		var err error
		if base, err = os.UserConfigDir(); err != nil {
			return nil
		}
	}
	p := filepath.Join(base, galeAppID)
	if st, err := fsx.Stat(p); err == nil && st.IsDir() {
		return []string{p}
	}
	return nil
}
