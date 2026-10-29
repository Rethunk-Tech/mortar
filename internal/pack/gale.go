package pack

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/winname"
	_ "modernc.org/sqlite"
)

// galeAppID is the folder Gale keeps its data in below the user's data directory
// (Kesomannen/gale src-tauri/src/util/path.rs, APP_GUID and default_app_data_dir).
const galeAppID = "com.kesomannen.gale"

// galeDB is Gale's database file (src-tauri/src/db/mod.rs:24).
const galeDB = "data.sqlite3"

// galeMod is one entry of a profile's mods JSON (src-tauri/src/profile/mod.rs:86 ProfileMod, :106 ThunderstoreMod,
// :400 LocalMod): a Thunderstore mod names its package as fullName, "Namespace-Name-Version"; a mod added from disk has
// no fullName, and its dependencies are packages named the same way.
type galeMod struct {
	FullName     string   `json:"fullName"`
	Enabled      *bool    `json:"enabled"`
	Name         string   `json:"name"`
	Author       string   `json:"author"`
	Version      string   `json:"version"`
	Dependencies []string `json:"dependencies"`
}

// galeRef is the package a "Namespace-Name-Version" names.
func galeRef(full string) (Ref, bool) {
	pkg, version, ok := deps.ThunderstoreRef(full)
	if !ok || !nativeID.MatchString(pkg) {
		return Ref{}, false
	}
	return Ref{Source: thunderstore, Native: pkg, Version: version}, true
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
	return GaleDB("")
}

// GaleDB is the path of the first Gale database found, looking in extra (the player's own folder, empty for none)
// before the default place, or "" when there is none.
func GaleDB(extra string) string {
	if dirs := GaleDataDirs(extra); len(dirs) > 0 {
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
	conn, err := sql.Open("sqlite", datadir.ReadOnlySQLiteURI(db))
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
		p.Path = reachablePath(p.Path, p.Slug, filepath.Dir(db))
		out = append(out, p)
	}
	return out, rows.Err()
}

// reachablePath is a profile's folder as this process can reach it. Gale records the absolute path it used, below
// <data dir>/<game slug>; when that path is not here (a Flatpak reaching the data folder through the document portal,
// or a data folder since moved) the same folder below the database's own folder is used.
func reachablePath(path, slug, dataDir string) string {
	if fsx.IsDir(path) {
		return path
	}
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	at := slices.Index(parts, slug)
	if at < 0 {
		return path
	}
	moved := filepath.Join(append([]string{dataDir}, parts[at:]...)...)
	if fsx.IsDir(moved) {
		return moved
	}
	return path
}

func (g Gale) find(ctx context.Context, path string) (GaleProfile, bool) {
	all, err := GaleProfiles(ctx, g.dbFile())
	if err != nil {
		return GaleProfile{}, false
	}
	for _, p := range all {
		if fsx.SamePath(p.Path, path) {
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

// Parse reads the profile's mods from the database and its BepInEx/config files. A mod Gale added from disk is a
// LocalMod of its folder, BepInEx/plugins/<name> (src-tauri/src/profile/import/local.rs:99-121), and the packages it
// depends on that the profile lacks are added to Packages.
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
	var deps []string
	for _, m := range mods {
		off := m.Enabled != nil && !*m.Enabled
		if m.FullName == "" {
			dir := filepath.Join(p.Path, "BepInEx", "plugins", m.Name)
			if winname.Valid(m.Name) && fsx.IsDir(dir) {
				d.Local = append(d.Local, LocalMod{Name: m.Name, Author: m.Author, Version: m.Version, Disabled: off, Dir: dir})
				deps = append(deps, m.Dependencies...)
			}
			continue
		}
		if ref, ok := galeRef(m.FullName); ok {
			ref.Disabled = off
			d.Packages = append(d.Packages, ref)
		}
	}
	for _, dep := range deps {
		if ref, ok := galeRef(dep); ok && !slices.ContainsFunc(d.Packages, func(r Ref) bool { return strings.EqualFold(r.Native, ref.Native) }) {
			d.Packages = append(d.Packages, ref)
		}
	}
	var err error
	if d.Configs, err = bepinexConfigs(p.Path); err != nil {
		return Draft{}, err
	}
	return d, nil
}

// GaleDataDirs lists Gale's data folders that exist on this machine: extra when it holds Gale's database (the player's
// own choice, empty for none), then <user data dir>/com.kesomannen.gale, with the user data dir as the dirs crate
// resolves it (XDG data home on Linux, %AppData% on Windows, Application Support on macOS).
func GaleDataDirs(extra string) []string {
	var out []string
	if extra != "" && fsx.IsFile(filepath.Join(extra, galeDB)) {
		out = append(out, extra)
	}
	var bases []string
	if runtime.GOOS == "linux" {
		// Inside the Flatpak XDG_DATA_HOME is the sandbox's own folder; the host's ~/.local/share is granted read-only.
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			bases = append(bases, xdg)
		}
		if home, err := os.UserHomeDir(); err == nil {
			bases = append(bases, sandbox.HostDataHome(home))
		}
	} else if base, err := os.UserConfigDir(); err == nil {
		bases = append(bases, base)
	}
	for _, base := range bases {
		p := filepath.Join(base, galeAppID)
		if st, err := fsx.Stat(p); err == nil && st.IsDir() && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// IsManagerFolder reports a folder r2modman or Gale could keep its data in: Gale's database, or r2modman's per-game
// folders each holding profiles.
func IsManagerFolder(dir string) bool {
	if fsx.IsFile(filepath.Join(dir, galeDB)) {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(entries, func(e os.DirEntry) bool { return fsx.IsDir(filepath.Join(dir, e.Name(), "profiles")) })
}
