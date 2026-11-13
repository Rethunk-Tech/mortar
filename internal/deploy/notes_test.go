package deploy

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSaveNotesReportsANewRescueOnce(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "j")
	rescue := filepath.Join(dir, "a.cfg.mortar-rescued")
	if err := os.WriteFile(rescue, []byte("edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	note := Note{Path: rescue, Kind: NoteRescued}
	fresh, err := saveNotes(journal, []Note{note})
	if err != nil || !slices.Equal(fresh, []string{rescue}) {
		t.Fatalf("fresh = %v %v", fresh, err)
	}
	again, err := saveNotes(journal, []Note{note})
	if err != nil || len(again) != 0 {
		t.Fatalf("again = %v %v", again, err)
	}
}

func TestPlaceSaveNotesTellsRescued(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "j")
	rescue := filepath.Join(dir, "a.cfg.mortar-rescued")
	if err := os.WriteFile(rescue, []byte("edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got []string
	p := Place{Rescued: func(paths []string) { got = paths }}
	if err := p.saveNotes(journal, []Note{{Path: rescue, Kind: NoteRescued}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{rescue}) {
		t.Fatalf("rescued = %v", got)
	}
}

func TestDiscardRescuedRemovesTheNotedFile(t *testing.T) {
	dir := t.TempDir()
	journal := filepath.Join(dir, "j")
	rescue := filepath.Join(dir, "a.cfg.mortar-rescued")
	if err := os.WriteFile(rescue, []byte("edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveNotes(journal, []Note{{Path: rescue, Kind: NoteRescued}}); err != nil {
		t.Fatal(err)
	}
	if err := DiscardRescued(journal, []string{rescue, filepath.Join(dir, "other")}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(rescue); !os.IsNotExist(err) {
		t.Fatalf("rescued file still there: %v", err)
	}
	if notes := Notes(journal); len(notes) != 0 {
		t.Fatalf("notes = %+v", notes)
	}
}
