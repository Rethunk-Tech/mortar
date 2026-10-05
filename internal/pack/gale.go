package pack

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	_ "modernc.org/sqlite"
)

// galeAppID is the folder Gale keeps its data in below the user's data directory
// (Kesomannen/gale src-tauri/src/util/path.rs, APP_GUID and default_app_data_dir).
const galeAppID = "com.kesomannen.gale"

// galeDB is Gale's database file (src-tauri/src/db/mod.rs:24).
const galeDB = "data.sqlite3"

// galeMod is one entry of a profile's mods JSON (src-tauri/src/profile/mod.rs:86 ProfileMod, :106 ThunderstoreMod): a
// Thunderstore mod names its package as fullName, "Namespace-Name-Version"; a mod added from disk has no fullName.
type galeMod struct {
	FullName string `json:"fullName"`
	Enabled  *bool  `json:"enabled"`
}

// Gale reads the profiles in Gale's database, data.sqlite3 in its data folder. Gale saves a profile with
// "INSERT OR REPLACE INTO profiles (id, name, path, game_slug, mods, ...)" (src-tauri/src/db/mod.rs:357, mods written
// as serde_json of the ProfileMod list at :375); the table is created in migrations/01-initial/up.sql. Its path
// column is the profile folder, whose BepInEx/config files are read too. It reads only.
type Gale struct {
	// GameBySlug maps Gale's game slug, which is the game's Thunderstore community key, to the catalog game id; nil or
	// a miss leaves the slug in Draft.Game.
	GameBySlug func(slug string) (string, bool)
	// DB is the database file; empty means the one in Gale's data folder.
	DB string
}

// GaleProfile is one profile row of Gale's database.
type GaleProfile struct {
	Name, Path, Slug string
	mods             string
}

func (g Gale) dbFile() string {
	if g.DB != "" {
		return g.DB
	}
	if dirs := GaleDataDirs(); len(dirs) > 0 {
		return filepath.Join(dirs[0], galeDB)
	}
	return ""
}

// GaleProfiles lists the profiles in the database file db; the database is opened read-only and immutable, so a
// running Gale is not disturbed (what it has not yet checkpointed from its write-ahead log is not seen).
func GaleProfiles(ctx context.Context, db string) ([]GaleProfile, error) {
	if db == "" {
		return nil, nil
	}
	if _, err := fsx.Stat(db); err != nil {
		return nil, nil
	}
	conn, err := sql.Open("sqlite", readOnlyURI(db))
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	rows, err := conn.QueryContext(ctx, "SELECT name, path, game_slug, mods FROM profiles ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("reading Gale's database: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []GaleProfile
	for rows.Next() {
		var p GaleProfile
		if err := rows.Scan(&p.Name, &p.Path, &p.Slug, &p.mods); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// readOnlyURI is SQLite's read-only URI for a database path. A Windows path ("C:/...") needs the leading slash too,
// or SQLite takes the drive for the URI's authority and refuses it.
func readOnlyURI(db string) string {
	p := filepath.ToSlash(db)
	if len(p) > 1 && p[1] == ':' {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p, RawQuery: "mode=ro&immutable=1"}).String()
}

func (g Gale) find(ctx context.Context, path string) (GaleProfile, bool) {
	all, err := GaleProfiles(ctx, g.dbFile())
	if err != nil {
		return GaleProfile{}, false
	}
	for _, p := range all {
		if filepath.Clean(p.Path) == filepath.Clean(path) {
			return p, true
		}
	}
	return GaleProfile{}, false
}

// ID names the format.
func (Gale) ID() string { return "gale-profile" }

// Detect accepts the folder of a profile in Gale's database.
func (g Gale) Detect(in Input) bool {
	_, ok := g.find(context.Background(), in.Path)
	return in.Path != "" && ok
}

// Parse reads the profile's mods from the database and its BepInEx/config files. Mods Gale added from disk carry no
// package name and are left out.
func (g Gale) Parse(ctx context.Context, in Input) (Draft, error) {
	p, ok := g.find(ctx, in.Path)
	if !ok {
		return Draft{}, errors.New("not a profile in Gale's database")
	}
	var mods []galeMod
	if err := json.Unmarshal([]byte(p.mods), &mods); err != nil {
		return Draft{}, fmt.Errorf("reading the mods of Gale profile %s: %w", p.Name, err)
	}
	d := Draft{Name: p.Name, Game: p.Slug}
	if g.GameBySlug != nil {
		if id, ok := g.GameBySlug(p.Slug); ok {
			d.Game = id
		}
	}
	for _, m := range mods {
		owner, rest, ok := strings.Cut(m.FullName, "-")
		name, version, ok2 := strings.Cut(rest, "-")
		if !ok || !ok2 || !nativeID.MatchString(owner+"-"+name) {
			continue
		}
		d.Packages = append(d.Packages, Ref{Source: thunderstore, Native: owner + "-" + name, Version: version, Disabled: m.Enabled != nil && !*m.Enabled})
	}
	var err error
	if d.Configs, err = bepinexConfigs(p.Path); err != nil {
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
