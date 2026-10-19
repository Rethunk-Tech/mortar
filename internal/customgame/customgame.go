// Package customgame keeps the games the player described for themselves: a game missing from the signed catalog,
// given a name, its executable and a loader. They live in one file in the data folder, join the catalog in use beside
// the signed games and are never uploaded.
package customgame

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// The loaders a custom game can pick, each the same catalog data a shipped game of that kind uses.
const (
	BepInEx5    = "bepinex5"
	BepInEx6    = "bepinex6"
	MelonLoader = "melonloader"
	// Shimloader is an Unreal Engine game modded through UE4SS; its Thunderstore community supplies the loader.
	Shimloader = "shimloader"
	// Folder is a game with no loader whose mods are files dropped in a Mods folder beside the executable.
	Folder = "folder"
)

// IDPrefix starts the id of every custom game, which keeps it apart from the catalog's slugs.
const IDPrefix = "custom-"

const (
	fileName = "custom-games.json"
	maxGames = 64
	maxName  = 60
)

// Entry is one custom game. Exe is the executable's file name, which marks the install folder.
type Entry struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Exe    string `json:"exe"`
	Loader string `json:"loader"`
	// Community is the game's Thunderstore community, which a Shimloader game takes its loader and mods from.
	Community string `json:"community,omitempty"`
}

type file struct {
	FormatVersion int     `json:"formatVersion,omitempty"`
	Games         []Entry `json:"games"`
}

var (
	notSlug   = regexp.MustCompile(`[^a-z0-9]+`)
	community = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
)

// shimloaderRoutes are the UE4SS mod folders of r2modman's Shimloader install rules, which every catalog game lists.
var shimloaderRoutes = []components.LoaderRoute{
	{Dir: "shimloader/mod", Track: components.TrackSubdir, Default: true},
	{Dir: "shimloader/pak", Extensions: []string{".pak"}, Track: components.TrackSubdir},
	{Dir: "shimloader/cfg"},
	{Dir: "shimloader/overlay", Track: components.TrackSubdir},
}

// Info is the catalog entry for the game, shaped like a shipped game of the same loader.
func (e Entry) Info() (components.GameInfo, error) {
	g := components.GameInfo{
		ID: e.ID, Name: e.Name, Enabled: true, Marker: e.Exe, Metadata: []string{}, Deploy: components.DeployProfile,
		Targets: []components.TargetDef{{ID: "profile", Root: "{profile}"}}, Sources: []components.GameSource{},
	}
	switch e.Loader {
	case BepInEx5:
		g.Loaders = []components.GameLoader{{ID: "bepinex5", Name: "BepInEx 5"}}
	case BepInEx6:
		g.Loaders = []components.GameLoader{{ID: "bepinex5", Name: "BepInEx 6 (IL2CPP)", Major: 6, Package: "BepInEx-BepInExPack_IL2CPP"}}
	case MelonLoader:
		g.Loaders = []components.GameLoader{{ID: "melonloader", Name: "MelonLoader", Package: "LavaGang-MelonLoader"}}
	case Shimloader:
		if !community.MatchString(e.Community) {
			return components.GameInfo{}, usererr.New(usererr.Invalid, "give the game's Thunderstore community, such as voices-of-the-void")
		}
		g.Loaders = []components.GameLoader{{ID: "shimloader", Name: "Shimloader", Package: "Thunderstore-unreal_shimloader", Routes: shimloaderRoutes}}
		g.Sources = []components.GameSource{{ID: "thunderstore", Key: e.Community}}
	case Folder:
		g.Loaders = []components.GameLoader{{ID: "folder"}}
		g.Targets = []components.TargetDef{{ID: "mods", Root: "{profile}/Mods"}}
		g.Paths = map[string]components.PathTemplate{"mods": {Windows: "{install}/Mods", Linux: "{install}/Mods"}}
	default:
		return components.GameInfo{}, usererr.New(usererr.Invalid, fmt.Sprintf("%q is not a loader a custom game can use", e.Loader))
	}
	return g, g.ValidateUser()
}

func path() (string, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// List reads the saved games; none when the file is absent.
func List() ([]Entry, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	b, err := fsx.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, usererr.Wrap(usererr.Damaged, fmt.Errorf("%s: %w", fileName, err))
	}
	if err := datadir.CheckVersion(f.FormatVersion); err != nil {
		return nil, err
	}
	if f.Games == nil {
		f.Games = []Entry{}
	}
	return f.Games, nil
}

func save(games []Entry) error {
	p, err := path()
	if err != nil {
		return err
	}
	return datadir.WriteVersioned(p, file{FormatVersion: datadir.FormatVersion, Games: games})
}

// Register makes the saved games part of the catalog in use. An entry that no longer validates is left out and logged
// by the caller's error; the rest still register.
func Register(games []Entry) error {
	infos := make([]components.GameInfo, 0, len(games))
	var errs []error
	for _, e := range games {
		info, err := e.Info()
		if err != nil {
			errs = append(errs, fmt.Errorf("custom game %q: %w", e.ID, err))
			continue
		}
		infos = append(infos, info)
	}
	components.SetUserGames(infos)
	return errors.Join(errs...)
}

// Load registers the saved games, for startup.
func Load() error {
	games, err := List()
	if err != nil {
		return err
	}
	return Register(games)
}

// Add saves a game named name, started by the executable at exe, and registers it. The install folder is the
// executable's folder, so the caller sets it as the game's folder.
func Add(name, exe, loader, community string) (Entry, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > maxName {
		return Entry{}, "", usererr.New(usererr.Invalid, "give the game a name of up to 60 characters")
	}
	if !filepath.IsAbs(exe) {
		return Entry{}, "", usererr.New(usererr.Invalid, "choose the game's executable by its full path")
	}
	if fi, err := os.Stat(exe); err != nil || fi.IsDir() {
		return Entry{}, "", usererr.New(usererr.Invalid, "that executable is not there")
	}
	games, err := List()
	if err != nil {
		return Entry{}, "", err
	}
	if len(games) >= maxGames {
		return Entry{}, "", usererr.New(usererr.Invalid, "too many custom games")
	}
	slug := strings.Trim(notSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		return Entry{}, "", usererr.New(usererr.Invalid, "the name needs letters or digits")
	}
	id := IDPrefix + slug
	for n := 2; taken(games, id); n++ {
		id = fmt.Sprintf("%s%s-%d", IDPrefix, slug, n)
	}
	e := Entry{ID: id, Name: name, Exe: filepath.Base(exe), Loader: loader, Community: community}
	if _, err := e.Info(); err != nil {
		return Entry{}, "", err
	}
	games = append(games, e)
	if err := save(games); err != nil {
		return Entry{}, "", err
	}
	return e, filepath.Dir(exe), Register(games)
}

func taken(games []Entry, id string) bool {
	if _, ok := components.Game(id); ok {
		return true
	}
	return slices.ContainsFunc(games, func(e Entry) bool { return e.ID == id })
}

// Remove forgets a custom game. Its profiles stay on disk, unlisted until the game is added again under the same name.
func Remove(id string) error {
	games, err := List()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(games, func(e Entry) bool { return e.ID == id })
	if i < 0 {
		return usererr.New(usererr.NotFound, "no such custom game")
	}
	games = slices.Delete(games, i, i+1)
	if err := save(games); err != nil {
		return err
	}
	return Register(games)
}
