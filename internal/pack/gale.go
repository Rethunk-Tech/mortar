package pack

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-modmanagers/gale"
	"github.com/Rethunk-Tech/go-modmanagers/r2modman"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/winname"
	_ "modernc.org/sqlite"
)

const galeAppID = gale.AppID

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
type GaleProfile = gale.Profile

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
		return gale.Database(dirs[0])
	}
	return ""
}

// GaleProfiles lists the profiles in the database file db; the database is opened read-only and immutable, so a
// running Gale is not disturbed (what it has not yet checkpointed from its write-ahead log is not seen).
func GaleProfiles(ctx context.Context, db string) ([]GaleProfile, error) {
	if db == "" {
		return nil, nil
	}
	return gale.Profiles(ctx, db)
}

func (g Gale) find(ctx context.Context, path string) (GaleProfile, bool) {
	all, err := GaleProfiles(ctx, g.dbFile())
	if err != nil {
		return GaleProfile{}, false
	}
	for _, p := range all {
		if safefs.SamePath(p.Path, path) {
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
	mods, err := p.Mods()
	if err != nil {
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
		off := !m.Enabled
		if m.FullName == "" {
			dir := filepath.Join(p.Path, "BepInEx", "plugins", m.Name)
			if winname.Valid(m.Name) && safefs.IsDir(dir) {
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
	if d.Configs, err = configFiles(p.Path); err != nil {
		return Draft{}, err
	}
	return d, nil
}

// GaleDataDirs lists Gale's data folders that exist on this machine: extra when it holds Gale's database (the player's
// own choice, empty for none), then <user data dir>/com.kesomannen.gale.
func GaleDataDirs(extra string) []string {
	var out []string
	if extra != "" && gale.HasDatabase(extra) {
		out = append(out, extra)
	}
	for _, d := range gale.DataDirs(gale.Bases()...) {
		if !slices.Contains(out, d) {
			out = append(out, d)
		}
	}
	return out
}

// IsManagerFolder reports a folder r2modman or Gale could keep its data in: Gale's database, or r2modman's per-game
// folders each holding profiles.
func IsManagerFolder(dir string) bool {
	return gale.HasDatabase(dir) || r2modman.IsDataFolder(dir)
}
