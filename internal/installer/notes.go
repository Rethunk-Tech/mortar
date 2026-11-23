package installer

import (
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// NotesFile is where a profile keeps what the Cyberpunk installer noted about each mod it placed, beside profile.json
// and outside the mods folder, so none of it is deployed into the game.
const NotesFile = "install-notes.json"

// ModNote is a Note and the mod folder it was made for.
type ModNote struct {
	Folder string `json:"folder"`
	Note
}

// ReadNotes returns the notes recorded in the profile folder.
func ReadNotes(profileDir string) []ModNote {
	var all []ModNote
	_, _ = datadir.ReadJSONOrQuarantine(filepath.Join(profileDir, NotesFile), &all)
	return all
}

// RecordNotes replaces the notes of one mod folder; no notes removes its old ones.
func RecordNotes(profileDir, folder string, notes []Note) error {
	all := slices.DeleteFunc(ReadNotes(profileDir), func(n ModNote) bool { return n.Folder == folder })
	for _, n := range notes {
		all = append(all, ModNote{Folder: folder, Note: n})
	}
	path := filepath.Join(profileDir, NotesFile)
	if len(all) == 0 && !datadir.Exists(path) {
		return nil
	}
	return datadir.WriteJSON(path, all)
}
