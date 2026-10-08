package profile

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/ids"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// GameModsProgressEvent is emitted with a GameModsProgress before each folder an import or move copies.
const GameModsProgressEvent = "profile:game-mods-progress"

// GameModsProgress says how far an import or move from the game's Mods folder has got: Done folders are finished and
// Name is the one being copied.
type GameModsProgress struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Name    string `json:"name"`
}

func (s *Store) reportGameModsProgress(game, id string, done, total int, name string) {
	if s.GameModsProgress != nil {
		s.GameModsProgress(GameModsProgress{Game: game, Profile: id, Done: done, Total: total, Name: name})
	}
}

// MoveGameMods moves the named top-level folders of modsDir (the game's own Mods folder) into the profile as local
// entries: each is copied into the store and added like Import from the game's Mods folder, then deleted from modsDir.
// A folder whose mods the profile already holds is skipped and left in place.
func (s *Store) MoveGameMods(ctx context.Context, game, id, modsDir string, folders []string) (GameModsResult, error) {
	return s.SyncGameMods(ctx, game, id, modsDir, folders, true, false)
}

// SyncGameMods brings the named top-level folders of modsDir into the profile. A folder whose mods the profile does
// not hold is added; one it holds is skipped, or with replace the profile's entry switches to the folder's copy.
// With move each folder that came in is then removed from modsDir; otherwise modsDir is not written. The whole sync
// is one history event.
func (s *Store) SyncGameMods(ctx context.Context, game, id, modsDir string, folders []string, move, replace bool) (GameModsResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return GameModsResult{}, err
	}
	if s.GameRunning != nil && s.GameRunning(game) {
		return GameModsResult{}, usererr.Wrap(usererr.Busy, errors.New("the game is running: close it before changing mods from its Mods folder"))
	}
	root, err := fsx.EvalSymlinks(modsDir)
	if err != nil {
		return GameModsResult{}, err
	}
	var slots []gameModSlot
	for _, folder := range folders {
		name, err := safeFolder(folder)
		if err != nil {
			return GameModsResult{}, err
		}
		child := filepath.Join(modsDir, name)
		if !datadir.RealDirUnder(root, child) {
			return GameModsResult{}, fmt.Errorf("%s is not a folder in the game's Mods folder", name)
		}
		if slot, keep := classifyFolder(child, name); keep {
			slots = append(slots, slot)
		}
	}
	resolveDuplicates(slots)
	total := 0
	for _, slot := range slots {
		if slot.ready {
			total++
		}
	}
	s.setHistoryQuiet(id, true)
	defer s.setHistoryQuiet(id, false)
	res := GameModsResult{Outcomes: []GameModOutcome{}}
	done := 0
	for _, slot := range slots {
		out := slot.outcome
		if slot.ready {
			s.reportGameModsProgress(game, id, done, total, slot.folder.label)
			done++
			out = s.syncFolder(ctx, game, id, slot.folder, move, replace)
		}
		switch out.Status {
		case outcomeImported:
			res.Imported++
		case outcomeSkipped:
			res.Skipped++
		case outcomeFailed:
			res.Failed++
		}
		res.Outcomes = append(res.Outcomes, out)
	}
	var change string
	if res.Imported > 0 {
		kind := ChangeImported
		if move {
			kind = ChangeMoved
		}
		if change, err = s.recordSnapshot(game, id, historyImported, HistoryEvent{Change: kind}, res.Imported); err != nil {
			return res, err
		}
		if err := s.RecordModsSnapshot(game, id); err != nil {
			return res, err
		}
	}
	res.Profile, err = s.read(game, id)
	res.Profile.LastChange = change
	return res, err
}

// syncFolder brings one folder into the profile (see SyncGameMods) and reports what happened to it.
func (s *Store) syncFolder(ctx context.Context, game, id string, f gameModFolder, move, replace bool) GameModOutcome {
	out := GameModOutcome{Name: f.label, Status: outcomeImported}
	fail := func(err error) GameModOutcome {
		out.Status, out.Reason = outcomeFailed, err.Error()
		if ie, ok := errors.AsType[*InstallError](err); ok {
			out.Reason = ie.Msg
		}
		return out
	}
	p, err := s.read(game, id)
	if err != nil {
		return fail(err)
	}
	heldKey, heldName := heldBy(p, f)
	switch {
	case heldKey == "":
		if _, _, err := s.importFolder(ctx, game, id, f); err != nil {
			return fail(err)
		}
	case !replace:
		return GameModOutcome{Name: f.label, Status: outcomeSkipped, Reason: "already in this profile as " + heldName}
	default:
		key, err := s.addResolvingLinks(ctx, game, f.dir)
		if err != nil {
			return fail(err)
		}
		src := f.source
		if _, err := s.moveTo(game, id, heldKey, key, &src); err != nil {
			if _, same := errors.AsType[*DuplicateError](err); same {
				return GameModOutcome{Name: f.label, Status: outcomeSkipped, Reason: "already up to date in this profile"}
			}
			return fail(err)
		}
	}
	if move {
		if err := s.removeFromGameMods(game, id, f.dir); err != nil {
			out.Reason = "added, but it could not be removed from the game's Mods folder: " + err.Error()
		}
	}
	return out
}

// heldBy returns the key and display name of the profile entry holding any mod of f, or "" when it holds none.
func heldBy(p Profile, f gameModFolder) (key, name string) {
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			if slices.ContainsFunc(f.mods, func(x manifest.Mod) bool { return mod.Equal(x.ModID(), m.ID) }) {
				return e.Key, m.Name
			}
		}
	}
	return "", ""
}

// removeFromGameMods deletes dir from the game's Mods folder by renaming it aside first: a rename is atomic, so a
// file held open by the game, an antivirus scan or the indexer leaves the folder whole instead of half deleted. The
// folder goes to Mortar's trash staging, or beside itself when that is another volume.
func (s *Store) removeFromGameMods(game, id, dir string) error {
	aside := filepath.Join(s.removedModsDir(game, id), "moved-"+ids.New(), filepath.Base(dir))
	err := os.MkdirAll(filepath.Dir(aside), 0o700)
	if err == nil {
		err = fsx.Rename(dir, aside)
	}
	if err != nil {
		aside = filepath.Join(filepath.Dir(dir), ".mortar-moved-"+ids.New())
		if err = fsx.Rename(dir, aside); err != nil {
			return err
		}
	}
	if err := fsx.RemoveAll(aside); err != nil {
		log.Printf("moved mods folder %s: clearing %s: %v", dir, aside, err)
	}
	return nil
}

// gameModsBucket is the settings dismissal bucket for game Mods folders the user chose not to move.
func gameModsBucket(game string) string { return "gameMods:" + game }

// NewGameModsFolders lists the mod folders in the game's own Mods folder that Mortar did not put there (SMAPI's
// bundled mods and Mortar's bridge are left out) and the user has not dismissed, as PreviewGameMods rows. Folders
// that hold no usable mod, or repeat a mod another folder holds, are not offered.
func (s *Service) NewGameModsFolders(game string) (GameModsPreview, error) {
	dir, err := s.gameModsDir(game)
	if err != nil {
		return GameModsPreview{}, err
	}
	slots, err := scanGameMods(dir)
	if err != nil {
		return GameModsPreview{}, err
	}
	dismissed := s.settings.Get().Dismissed[gameModsBucket(game)]
	slots = slices.DeleteFunc(slots, func(sl gameModSlot) bool {
		return !sl.ready || slices.Contains(dismissed, filepath.Base(sl.folder.dir))
	})
	return previewFrom(slots), nil
}

// DismissGameModsFolder stops offering folder, a top-level folder of the game's Mods folder, for moving.
func (s *Service) DismissGameModsFolder(game, folder string) error {
	name, err := safeFolder(folder)
	if err != nil {
		return err
	}
	return s.settings.AppendDismissed(gameModsBucket(game), name)
}

// UndismissGameModsFolders offers folders again after DismissGameModsFolder stopped offering them.
func (s *Service) UndismissGameModsFolders(game string, folders []string) error {
	names := make([]string, 0, len(folders))
	for _, f := range folders {
		name, err := safeFolder(f)
		if err != nil {
			return err
		}
		names = append(names, name)
	}
	return s.settings.RemoveDismissed(gameModsBucket(game), names...)
}

// SyncGameModsFolders brings the given top-level folders of the game's Mods folder into the profile: new ones are
// added and, for a folder the profile already holds, the profile's copy is replaced by the folder's. With move each
// folder that came in is then removed from the game folder; otherwise the game folder is not written.
func (s *Service) SyncGameModsFolders(ctx context.Context, game, id string, folders []string, move bool) (GameModsResult, error) {
	dir, err := s.gameModsDir(game)
	if err != nil {
		return GameModsResult{}, err
	}
	return s.store.SyncGameMods(ctx, game, id, dir, folders, move, true)
}

// MoveGameModsFolders moves the given top-level folders of the game's Mods folder into the profile.
func (s *Service) MoveGameModsFolders(ctx context.Context, game, id string, folders []string) (GameModsResult, error) {
	dir, err := s.gameModsDir(game)
	if err != nil {
		return GameModsResult{}, err
	}
	return s.store.MoveGameMods(ctx, game, id, dir, folders)
}

// GameModChange is one mod both the game's Mods folder and the profile hold, at different version strings.
type GameModChange struct {
	ID     mod.ID `json:"id"`
	Name   string `json:"name"`
	Folder string `json:"folder"`
	// FolderVersion and ProfileVersion are the two copies' versions.
	FolderVersion  string `json:"folderVersion"`
	ProfileVersion string `json:"profileVersion"`
	// Newer is "folder" or "profile" when the versions order, and "" when they do not.
	Newer string `json:"newer"`
}

// GameModsDiff is the game's Mods folder compared with one profile, mod by UniqueID.
type GameModsDiff struct {
	// Missing are folder mods the profile lacks; Folder is the absolute top-level folder.
	Missing   []GameModPreview `json:"missing"`
	Different []GameModChange  `json:"different"`
	// Unreadable are folders Mortar cannot import (an invalid manifest, no mod); they never count toward the banner.
	Unreadable []GameModPreview `json:"unreadable"`
	// Same counts mods at the same version in both; ProfileOnly counts the profile's mods with no folder copy.
	Same        int `json:"same"`
	ProfileOnly int `json:"profileOnly"`
}

// GameModsDiff compares the game's Mods folder with the profile. Dismissed folders are left out of every list.
func (s *Service) GameModsDiff(game, id string) (GameModsDiff, error) {
	dir, err := s.gameModsDir(game)
	if err != nil {
		return GameModsDiff{}, err
	}
	return s.store.GameModsDiff(game, id, dir, s.settings.Get().Dismissed[gameModsBucket(game)])
}

// GameModsDiff scans modsDir like NewGameModsFolders and compares each mod with the profile's.
func (s *Store) GameModsDiff(game, id, modsDir string, dismissed []string) (GameModsDiff, error) {
	slots, err := scanGameMods(modsDir)
	if err != nil {
		return GameModsDiff{}, err
	}
	p, err := s.load(game, id)
	if err != nil {
		return GameModsDiff{}, err
	}
	scheme := gamereg.VersionScheme(game)
	held := indexMergeable(p)
	out := GameModsDiff{Missing: []GameModPreview{}, Different: []GameModChange{}, Unreadable: []GameModPreview{}}
	inFolder := map[string]bool{}
	for _, sl := range slots {
		hidden := slices.Contains(dismissed, filepath.Base(sl.folder.dir))
		if !sl.ready {
			if sl.outcome.Status == outcomeFailed && !hidden {
				out.Unreadable = append(out.Unreadable, GameModPreview{
					Name: sl.outcome.Name, Status: outcomeFailed, Reason: sl.outcome.Reason, Folder: sl.folder.dir,
				})
			}
			continue
		}
		for _, m := range sl.folder.mods {
			k := m.ModID().Fold()
			if manifest.LoaderManaged(m.ModID()) {
				continue
			}
			inFolder[k] = true
			if hidden {
				continue
			}
			have, ok := held[k]
			switch {
			case !ok:
				out.Missing = append(out.Missing, previewMod(sl.folder, m))
			case have.Version == m.Version:
				out.Same++
			default:
				c, ordered := deps.Compare(scheme, m.Version, have.Version)
				ch := GameModChange{
					ID: m.ModID(), Name: cmp.Or(m.Name, m.ModID().Local()), Folder: sl.folder.dir,
					FolderVersion: m.Version, ProfileVersion: have.Version,
				}
				switch {
				case ordered && c > 0:
					ch.Newer = "folder"
				case ordered && c < 0:
					ch.Newer = "profile"
				}
				out.Different = append(out.Different, ch)
			}
		}
	}
	for k := range held {
		if !inFolder[k] {
			out.ProfileOnly++
		}
	}
	return out, nil
}
