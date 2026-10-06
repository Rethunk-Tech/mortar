// Package game is the registry of supported games: the Game interface, its Stardew implementation and display-only entries.
package game

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"

	_ "github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher" // registers the frameworks LoaderRef reads
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	_ "github.com/Rethunk-Tech/mortar/internal/loader/all"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/steam"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const artPrefix = "/steam-art/"

// ArtURL is where the frontend loads Steam's hero art for appID from; ArtMiddleware serves it.
func ArtURL(appID string) string { return artPrefix + appID }

// Game is what a game needs code for: its identity and install folder. How it is modded and started belongs to its
// loaders and stores.
type Game interface {
	Identity
	Installs
}

// Identity names a game and where its process and mods come from.
type Identity interface {
	ID() string
	Name() string
	SteamAppID() string
	ModSources() []string
	// GameProcesses are the game's own executables; any of them running blocks a loader install, whoever started it.
	GameProcesses() []string
}

// Installs finds and validates a game's install folder.
type Installs interface {
	// ValidInstall reports why dir is not this game's install folder, or nil.
	ValidInstall(dir string) error
}

// Loaders are the game's loader drivers in catalog order; a catalog entry without a registered driver is left out.
func Loaders(id string) []loader.Loader {
	g, _ := catalogGame(id)
	return loader.For(g)
}

// VersionScheme is the scheme the game's components are versioned in: its first loader's that names one, else opaque.
func VersionScheme(id string) string {
	for _, l := range Loaders(id) {
		if v, ok := l.(loader.VersionScheme); ok {
			return v.VersionScheme()
		}
	}
	return deps.Opaque
}

// PrimaryLoader is the game's first loader driver.
func PrimaryLoader(id string) (loader.Loader, bool) {
	all := Loaders(id)
	if len(all) == 0 {
		return nil, false
	}
	return all[0], true
}

// LoaderOf is the game's loader with this id; "" is its primary loader.
func LoaderOf(id, loaderID string) (loader.Loader, bool) {
	if loaderID == "" {
		return PrimaryLoader(id)
	}
	for _, l := range Loaders(id) {
		if l.ID() == loaderID {
			return l, true
		}
	}
	return nil, false
}

// LoaderStatus is the state of the game's loader (loaderID, "" for the primary) in the install in dir. recorded is the
// version Mortar installed per game and loader (settings.Loaders), which outranks the version the install itself reports.
func LoaderStatus(id, loaderID, dir string, recorded map[string]string) (loader.Status, error) {
	l, ok := LoaderOf(id, loaderID)
	if !ok {
		return loader.Status{}, fmt.Errorf("game %q has no loader %q", id, loaderID)
	}
	st, err := l.Status(loader.Target{Game: id, InstallDir: dir})
	if err != nil {
		return loader.Status{}, err
	}
	if st.Installed || st.Broken {
		st.Version = cmp.Or(recorded[settings.LoaderKey(id, l.ID())], st.Version)
	}
	return st, nil
}

// ParseLaunchOptions splits a profile's extra launch arguments for the game, refusing the flags its loader sets itself.
func ParseLaunchOptions(id, loaderID, options string) ([]string, error) {
	var denied []string
	if l, ok := LoaderOf(id, loaderID); ok {
		if d, ok := l.(loader.DeniedArgs); ok {
			denied = d.DeniedArgs()
		}
	}
	return launchplan.ParseArgs(options, denied)
}

// LogFile is the game's first loader's log, which outlives the game.
func LogFile(id string) (string, error) {
	l, _ := PrimaryLoader(id)
	logs, ok := l.(loader.WithLogs)
	if !ok {
		return "", fmt.Errorf("game %q has no loader log", id)
	}
	return logs.Path(loader.ProfileView{})
}

// LoaderName is how the catalog names the game's loader with this id; "" is its first.
func LoaderName(id, loaderID string) string {
	g, ok := catalogGame(id)
	if !ok {
		return ""
	}
	for _, l := range g.Loaders {
		if loaderID == "" || l.ID == loaderID {
			return l.Name
		}
	}
	return ""
}

// ProcessNames are the executables a running game is found by: its own and its loaders'.
func ProcessNames(g Game) []string {
	names := slices.Clone(g.GameProcesses())
	for _, l := range Loaders(g.ID()) {
		if p, ok := l.(loader.ProcessNames); ok {
			names = append(names, p.ProcessNames()...)
		}
	}
	return names
}

// coded are the games whose behaviour the catalog cannot describe; every other catalog game is a catalogOnly.
var coded = []Game{stardewValley{"stardew"}}

// ConfigureComponents selects the verified component manifest for the whole process (components.Use).
func ConfigureComponents(client *components.Client) { components.Use(client) }

// Catalog returns every game the component manifest lists, enabled or coming later. It is the bundled manifest's
// until a verified one that lists games is selected.
func Catalog() []components.GameInfo { return components.Games() }

// CatalogSerial is the serial of the component manifest in use: the verified one once selected, else the bundled one.
func CatalogSerial() uint64 {
	if c := components.Active(); c != nil {
		if m := c.Manifest(); m.Serial > 0 {
			return m.Serial
		}
	}
	m, _ := components.BundledManifest()
	return m.Serial
}

func catalogGame(id string) (components.GameInfo, bool) {
	return components.Game(id)
}

// ByNexusDomain is the enabled game whose Nexus domain is domain.
func ByNexusDomain(domain string) (string, bool) {
	for _, g := range Catalog() {
		if g.Enabled && g.NexusDomain() != "" && g.NexusDomain() == domain {
			return g.ID, true
		}
	}
	return "", false
}

// ByR2modmanFolder is the catalog game that r2modman keeps under the folder name, enabled or not.
func ByR2modmanFolder(name string) (string, bool) {
	for _, g := range Catalog() {
		if g.R2modmanFolder != "" && g.R2modmanFolder == name {
			return g.ID, true
		}
	}
	return "", false
}

// NexusTitle is how Nexus knows the game with this id, from the catalog's nexus source.
func NexusTitle(id string) (nexus.Title, error) {
	g, ok := catalogGame(id)
	if !ok || g.NexusDomain() == "" {
		return nexus.Title{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("game %q has no Nexus domain", id))
	}
	return nexus.Title{Domain: g.NexusDomain(), ID: g.NexusID()}, nil
}

// Find returns the game with this id: its coded implementation, else the catalog's entry; nil when unlisted.
func Find(id string) Game {
	for _, g := range coded {
		if g.ID() == id {
			return g
		}
	}
	if _, ok := catalogGame(id); ok {
		return catalogOnly(id)
	}
	return nil
}

// Implemented lists the ids of enabled catalog games that have a registered implementation.
func Implemented() []string {
	var ids []string
	for _, g := range Catalog() {
		if g.Enabled && Find(g.ID) != nil {
			ids = append(ids, g.ID)
		}
	}
	return ids
}

// Require returns the implemented game with this id, or a NotFound error.
func Require(id string) (Game, error) {
	g := Find(id)
	if g == nil {
		return nil, usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", id))
	}
	return g, nil
}

// Valid reports whether id names a listed game.
func Valid(id string) bool {
	return slices.ContainsFunc(Catalog(), func(g components.GameInfo) bool { return g.ID == id })
}

// File is the path of a game's JSON file below root, for stores that keep one file per game.
func File(root, id string) (string, error) {
	if !Valid(id) {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("unknown game %q", id))
	}
	return filepath.Join(root, id+".json"), nil
}

// listedApp returns the listing's own copy of appID, so values built from it never carry request text.
func listedApp(appID string) (string, bool) {
	if _, err := strconv.ParseUint(appID, 10, 32); err != nil {
		return "", false
	}
	for _, g := range Catalog() {
		if id := g.SteamAppID(); id == appID {
			return id, true
		}
	}
	return "", false
}

// heroCDN is Steam's public copy of a game's hero image, the same file Steam caches locally.
var heroCDN = "https://cdn.cloudflare.steamstatic.com/steam/apps/%s/library_hero.jpg"

// maxHero bounds a fetched hero image; Steam's are a few hundred KB.
const maxHero = 8 << 20

// ArtMiddleware serves GET /steam-art/<appid> for listed games only, from Steam's cached hero image, else from
// Steam's public copy fetched here: the webview does not follow a redirect from the app's own scheme to https.
func ArtMiddleware(home string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := strings.CutPrefix(r.URL.Path, artPrefix)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			listed, ok := listedApp(id)
			if r.Method != http.MethodGet || !ok {
				http.NotFound(w, r)
				return
			}
			path := ""
			for _, one := range steam.LocateAll(home) {
				path = one.HeroArt(id)
				if path != "" {
					break
				}
			}
			if path == "" {
				serveRemoteArt(w, r, fmt.Sprintf(heroCDN, listed))
				return
			}
			art, err := fsx.Open(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer func() { _ = art.Close() }()
			info, err := art.Stat()
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			http.ServeContent(w, r, "", info.ModTime(), art)
		})
	}
}

func serveRemoteArt(w http.ResponseWriter, r *http.Request, url string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "max-age=86400")
	_, _ = io.Copy(w, io.LimitReader(resp.Body, maxHero))
}
