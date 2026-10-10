package deploy

import (
	"encoding/json"
	"os"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Note kinds: a file that stays in the shared folder because it was too large to adopt, and a file that holds changed
// bytes Purge could not put into a profile.
const (
	NoteTooLarge = "toolarge"
	NoteRescued  = "rescued"
)

// Note is a file Purge left in the shared folder that the player should hear about.
type Note struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Kind string `json:"kind"`
}

// notesPath sits beside the journal folder, which Purge removes, so a note outlives the launch until its file is gone.
func notesPath(journalDir string) string { return journalDir + ".shared.json" }

// saveNotes merges notes into the install's record.
func saveNotes(journalDir string, notes []Note) error {
	if len(notes) == 0 || journalDir == "" {
		return nil
	}
	all := Notes(journalDir)
	for _, n := range notes {
		if !slices.ContainsFunc(all, func(x Note) bool { return x.Path == n.Path && x.Kind == n.Kind }) {
			all = append(all, n)
		}
	}
	return writeNotes(journalDir, all)
}

func writeNotes(journalDir string, notes []Note) error {
	if len(notes) == 0 {
		err := fsx.Remove(notesPath(journalDir))
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	b, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	return datadir.WriteFile(notesPath(journalDir), b, 0o600)
}

// Notes lists the files an earlier purge left in the shared folder that are still there, with their current size; a
// note whose file is gone is dropped.
func Notes(journalDir string) []Note {
	b, err := fsx.ReadFile(notesPath(journalDir))
	if err != nil {
		return nil
	}
	var all, live []Note
	if json.Unmarshal(b, &all) != nil {
		return nil
	}
	for _, n := range all {
		if st, err := os.Lstat(n.Path); err == nil && st.Mode().IsRegular() {
			n.Size = st.Size()
			live = append(live, n)
		}
	}
	if len(live) != len(all) {
		_ = writeNotes(journalDir, live)
	}
	return live
}
