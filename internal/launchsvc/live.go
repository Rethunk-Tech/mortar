package launchsvc

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// LiveEvent carries a LiveRun while the game runs.
const LiveEvent = "launch:live"

// askLiveFor bounds one question to the companion, which answers on localhost, so a stuck game cannot hold up the
// next one.
const askLiveFor = 500 * time.Millisecond

// LiveRun is what the running game's companion reports, for the Running header and the mod list's loaded badges.
type LiveRun struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Scene   string `json:"scene"`
	// Mods are the profile's packages that ship plugins, each loaded when any of its plugins is among the game's.
	Mods []LiveMod `json:"mods"`
}

type LiveMod struct {
	ID     mod.ID `json:"id"`
	Loaded bool   `json:"loaded"`
}

// followRunning asks the companion at the running-state poll's pace for as long as the run lasts, keeping the game
// version and announcing the scene and which packages loaded.
func (s *Service) followRunning(ctx context.Context, g game.Game) {
	tick := time.NewTicker(pollEvery)
	defer tick.Stop()
	var packages func() livePackages
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		st := s.current(g)
		if st.State == Idle {
			return
		}
		if st.State != Running || st.Profile == "" {
			continue
		}
		l, _ := s.loaderOf(g.ID(), st.Profile)
		live, ok := l.(loader.RunningState)
		if !ok || s.profiles == nil {
			return
		}
		if packages == nil {
			packages = s.livePackages(g.ID(), st.Profile)
		}
		run, ok := s.askLive(ctx, g, st, live, packages)
		if ok {
			s.emit(LiveEvent, run)
		}
	}
}

// askLive asks the companion once; the first version it reports is kept on the session and recorded as the version
// last played.
func (s *Service) askLive(ctx context.Context, g game.Game, st Status, live loader.RunningState, packages func() livePackages) (LiveRun, bool) {
	dir, err := s.profiles.ProfileDir(g.ID(), st.Profile)
	if err != nil {
		return LiveRun{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, askLiveFor)
	defer cancel()
	got, err := live.RunningState(ctx, loader.ProfileView{Game: g.ID(), Dir: dir})
	if err != nil {
		return LiveRun{}, false
	}
	if got.GameVersion != "" {
		s.noteGameVersion(g, st.Profile, got.GameVersion)
	}
	return LiveRun{Game: g.ID(), Profile: st.Profile, Scene: got.Scene, Mods: liveMods(packages(), got.Plugins)}, true
}

func (s *Service) noteGameVersion(g game.Game, profileID, v string) {
	key := keyOf(g)
	s.mu.Lock()
	sess, ok := s.logs[key]
	fresh := ok && sess.profile == profileID && sess.gameVersion == ""
	if fresh {
		sess.gameVersion = v
		s.logs[key] = sess
	}
	s.mu.Unlock()
	if fresh && s.settings != nil && !profile.IsScratch(profileID) {
		_, _ = s.settings.RecordLastPlayed(g.ID(), profileID, time.Now(), v)
	}
}

// livePackages is what tells which of the profile's packages a running game loaded: each plugin the enabled
// packages declare in their DLLs, and the entry key that laid out each file below the profile's root.
type livePackages struct {
	plugins []declaredPlugin
	files   map[string]string
}

// declaredPlugin is one [BepInPlugin] in an enabled package's DLLs.
type declaredPlugin struct {
	ID                 mod.ID
	Key, GUID, Version string
}

// livePackages reads the profile's packages once per run, and only once the game answers.
func (s *Service) livePackages(gameID, profileID string) func() livePackages {
	return sync.OnceValue(func() livePackages {
		var out livePackages
		out.files, _ = s.profiles.PackageFileOwners(gameID, profileID)
		installed, err := s.profiles.Installed(gameID, profileID)
		if err != nil {
			return out
		}
		for _, im := range installed {
			if !im.Enabled || im.Folder == "" {
				continue
			}
			for _, pl := range dotnet.PluginsIn(im.Folder) {
				out.plugins = append(out.plugins, declaredPlugin{ID: im.ModID(), Key: im.Key, GUID: pl.GUID, Version: pl.Version})
			}
		}
		return out
	})
}

// pluginsFolder is where BepInEx's chainloader looks for plugins, below the profile's root; a LivePlugin's Location
// is below it.
const pluginsFolder = "BepInEx/plugins/"

// liveMods marks each package that ships plugins loaded when BepInEx loaded one of them. A plugin is the package's
// whose copy of the file BepInEx loaded, when the companion names the file; otherwise the one declaring its GUID at
// the version that loaded. Two packages can declare one GUID: BepInEx keeps the newest version, which the version
// match finds, and between equal versions the first file its folder walk meets, an order only the file's location
// tells, so a companion that names none leaves both copies marked loaded.
func liveMods(pkgs livePackages, loaded []loader.LivePlugin) []LiveMod {
	byID := map[mod.ID]bool{}
	for _, d := range pkgs.plugins {
		byID[d.ID] = false
	}
	for _, pl := range loaded {
		if key, ok := pkgs.files[pluginsFolder+pl.Location]; ok && pl.Location != "" {
			for _, d := range pkgs.plugins {
				if d.Key == key && strings.EqualFold(d.GUID, pl.GUID) {
					byID[d.ID] = true
				}
			}
			continue
		}
		for _, d := range pkgs.plugins {
			if strings.EqualFold(d.GUID, pl.GUID) && sameVersion(d.Version, pl.Version) {
				byID[d.ID] = true
			}
		}
	}
	out := make([]LiveMod, 0, len(byID))
	for id, loaded := range byID {
		out = append(out, LiveMod{ID: id, Loaded: loaded})
	}
	slices.SortFunc(out, func(a, b LiveMod) int { return strings.Compare(string(a.ID), string(b.ID)) })
	return out
}

// sameVersion compares a [BepInPlugin] version as written with the one BepInEx reports, System.Version's form, which
// drops leading zeros ("1.02" is 1.2).
func sameVersion(declared, loaded string) bool {
	a, b := strings.Split(declared, "."), strings.Split(loaded, ".")
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, errA := strconv.Atoi(strings.TrimSpace(a[i]))
		y, errB := strconv.Atoi(b[i])
		if errA != nil || errB != nil || x != y {
			return false
		}
	}
	return true
}
