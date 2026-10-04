package profile

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/ids"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

// MoveGameMods moves the named top-level folders of modsDir (the game's own Mods folder) into the profile as local
// entries: each is copied into the store and added like Import from the game's Mods folder, then deleted from modsDir.
// A folder whose mods the profile already holds is skipped and left in place.
func (s *Store) MoveGameMods(game, id, modsDir string, folders []string) (GameModsResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return GameModsResult{}, err
	}
	if s.GameRunning != nil && s.GameRunning(game) {
		return GameModsResult{}, usererr.Wrap(usererr.Busy, errors.New("the game is running: close it before moving mods out of its Mods folder"))
	}
	root, err := filepath.EvalSymlinks(modsDir)
	if err != nil {
		return GameModsResult{}, err
	}
	p, err := s.read(game, id)
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
	held := func(f gameModFolder) string {
		for _, e := range p.Entries {
			for _, m := range e.Mods {
				if slices.ContainsFunc(f.mods, func(x manifest.Mod) bool { return SameID(x.UniqueID, m.UniqueID) }) {
					return m.Name
				}
			}
		}
		return ""
	}
	s.setHistoryQuiet(id, true)
	defer s.setHistoryQuiet(id, false)
	res := GameModsResult{Outcomes: []GameModOutcome{}}
	for _, slot := range slots {
		out := slot.outcome
		switch {
		case !slot.ready:
		case held(slot.folder) != "":
			out = GameModOutcome{Name: slot.folder.label, Status: outcomeSkipped, Reason: "already in this profile as " + held(slot.folder)}
		default:
			out = GameModOutcome{Name: slot.folder.label, Status: outcomeImported}
			if _, err := s.importFolder(game, id, slot.folder); err != nil {
				out.Status, out.Reason = outcomeFailed, err.Error()
				if ie, ok := errors.AsType[*InstallError](err); ok {
					out.Reason = ie.Msg
				}
			} else if err := s.removeFromGameMods(game, id, slot.folder.dir); err != nil {
				out.Reason = "added, but it could not be removed from the game's Mods folder: " + err.Error()
			}
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
	if res.Imported > 0 {
		label := fmt.Sprintf("Moved %d mods from the game's Mods folder", res.Imported)
		if err := s.recordSnapshot(game, id, historyImported, label, res.Imported); err != nil {
			return res, err
		}
		if err := s.RecordModsSnapshot(game, id); err != nil {
			return res, err
		}
	}
	res.Profile, err = s.read(game, id)
	return res, err
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

// MoveGameModsFolders moves the given top-level folders of the game's Mods folder into the profile.
func (s *Service) MoveGameModsFolders(game, id string, folders []string) (GameModsResult, error) {
	dir, err := s.gameModsDir(game)
	if err != nil {
		return GameModsResult{}, err
	}
	return s.store.MoveGameMods(game, id, dir, folders)
}
