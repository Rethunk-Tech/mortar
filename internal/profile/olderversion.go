package profile

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

var digits = regexp.MustCompile(`^[0-9]+$`)

// OlderVersion is what the "Get an older version" dialog needs: the game's Steam app and the depots it has installed.
type OlderVersion struct {
	AppID  string           `json:"appId"`
	Depots []steam.Depot    `json:"depots"`
	Builds []game.PastBuild `json:"builds"` // manifests Mortar saw installed, newest first
}

// OlderVersionState is whether Steam has finished writing a depot download.
type OlderVersionState struct {
	Dir   string `json:"dir"`
	Ready bool   `json:"ready"`
}

// downloadDepotCommand is what the player pastes into Steam's console.
func downloadDepotCommand(appID, depotID, manifest string) (string, error) {
	for _, v := range []string{appID, depotID, manifest} {
		if !digits.MatchString(v) {
			return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a number: ids and manifest ids are digits only", v))
		}
	}
	return "download_depot " + appID + " " + depotID + " " + manifest, nil
}

// versionFolder is where a copied older version lives: one folder per manifest below the data folder.
func versionFolder(dataDir, gameID, manifest string) string {
	return filepath.Join(dataDir, "versions", gameID, manifest)
}

// steamInstall finds the selected install's Steam app and library, the place Steam's console writes depots to.
func (s *Service) steamInstall(gameID string) (appID, library, dir string, err error) {
	info, ok := components.Game(gameID)
	if !ok || info.SteamAppID() == "" {
		return "", "", "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not a Steam game", gameID))
	}
	inst, err := game.ResolveInstall(s.home, s.settings.Get(), gameID, "")
	if err != nil {
		return "", "", "", err
	}
	if library = game.LibraryOf(inst.Dir); library == "" {
		return "", "", "", usererr.Wrap(usererr.Invalid, fmt.Errorf("the selected install of %s is not in a Steam library", gameID))
	}
	return info.SteamAppID(), library, inst.Dir, nil
}

// CanGetOlderVersion reports whether the game's selected install is a Steam one, the only kind Steam's console can
// download an older version of.
func (s *Service) CanGetOlderVersion(gameID string) bool {
	_, _, _, err := s.steamInstall(gameID)
	return err == nil
}

// OlderVersionDepots lists the depots of the game's Steam install.
func (s *Service) OlderVersionDepots(gameID string) (OlderVersion, error) {
	appID, library, dir, err := s.steamInstall(gameID)
	if err != nil {
		return OlderVersion{}, err
	}
	got, err := steam.InstalledDepots(library, appID)
	if err != nil {
		return OlderVersion{}, err
	}
	game.RecordBuildsIn(s.store.dataDir, gameID, appID, dir, time.Now())
	return OlderVersion{AppID: appID, Depots: got, Builds: game.BuildHistory(s.store.dataDir, gameID, dir)}, nil
}

// DownloadDepotCommand is the Steam console line that downloads the manifest of a depot.
func (s *Service) DownloadDepotCommand(gameID, depotID, manifest string) (string, error) {
	info, _ := components.Game(gameID)
	return downloadDepotCommand(info.SteamAppID(), depotID, manifest)
}

// OlderVersionState reports whether Steam's console has finished the depot download, which is when the folder holds the game.
func (s *Service) OlderVersionState(gameID, depotID string) (OlderVersionState, error) {
	appID, library, _, err := s.steamInstall(gameID)
	if err != nil {
		return OlderVersionState{}, err
	}
	if !digits.MatchString(depotID) {
		return OlderVersionState{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a depot id", depotID))
	}
	dir := steam.DepotDownloadDir(library, appID, depotID)
	g := game.Find(gameID)
	return OlderVersionState{Dir: dir, Ready: g != nil && g.ValidInstall(dir) == nil}, nil
}

// AdoptOlderVersion copies the finished depot download into Mortar's versions folder and points the profile at the
// copy. Steam's content folder is left as it is.
func (s *Service) AdoptOlderVersion(ctx context.Context, gameID, id, depotID, manifest string) (Profile, error) {
	st, err := s.OlderVersionState(gameID, depotID)
	if err != nil {
		return Profile{}, err
	}
	if !st.Ready {
		return Profile{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("the Steam download in %s is not finished", st.Dir))
	}
	if !digits.MatchString(manifest) {
		return Profile{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a manifest id", manifest))
	}
	dst := versionFolder(s.store.dataDir, gameID, manifest+"-"+time.Now().Format("20060102"))
	if _, err := safefs.CopyTree(st.Dir, dst, safefs.CopyOptions{}); err != nil {
		return Profile{}, err
	}
	return s.store.SetInstall(gameID, id, game.FolderPin(dst))
}

// SetGameFolder runs the profile from dir, an old copy of the game; "" goes back to the selected install.
func (s *Service) SetGameFolder(gameID, id, dir string) (Profile, error) {
	if strings.TrimSpace(dir) == "" {
		return s.store.SetInstall(gameID, id, "")
	}
	if err := game.ValidateFolder(gameID, dir); err != nil {
		return Profile{}, usererr.Wrap(usererr.Invalid, err)
	}
	return s.store.SetInstall(gameID, id, game.FolderPin(dir))
}

// SetModsUpTo makes the profile install and update mods at the newest version released on or before date
// (YYYY-MM-DD); "" lifts the limit.
func (s *Service) SetModsUpTo(gameID, id, date string) (Profile, error) {
	if date != "" && !source.ValidCutoff(date) {
		return Profile{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a date (YYYY-MM-DD)", date))
	}
	return s.store.update(gameID, id, func(p *Profile, _ string) error {
		p.ModsUpTo = date
		return nil
	})
}

// ModsUpTo is the profile's cutoff, zero when it has none.
func (s *Store) ModsUpTo(gameID, id string) time.Time {
	p, err := s.read(gameID, id)
	if err != nil {
		return time.Time{}
	}
	return source.ParseCutoff(p.ModsUpTo)
}
