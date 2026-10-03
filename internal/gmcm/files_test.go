package gmcm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

func TestReadWriteCaptureAndPending(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	field := "Volume"
	menu := Capture{
		Schema: Schema,
		Mod:    ModInfo{ID: "Example.Mod", Name: "Example", Version: "1.0.0"},
		Pages: []Page{{
			ID:    "main",
			Title: "Main",
			Options: []Option{{
				Index:   0,
				Kind:    "int",
				FieldID: &field,
				Name:    "Volume",
				Value:   4.0,
				Min:     new(0.0),
				Max:     new(10.0),
			}},
		}},
	}
	if err := os.MkdirAll(filepath.Join(dir, captureDir), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteJSON(CapturePath(dir, "Example.Mod"), menu); err != nil {
		t.Fatal(err)
	}
	got, err := ReadCapture(dir, "Example.Mod")
	if err != nil {
		t.Fatal(err)
	}
	if got.Mod.ID != "Example.Mod" || len(got.Pages) != 1 || got.Pages[0].Options[0].Name != "Volume" {
		t.Fatalf("capture: %+v", got)
	}
	if err := datadir.WriteJSON(filepath.Join(dir, captureDir, indexName), Index{
		Schema: Schema,
		Mods:   []IndexMod{{ID: "Example.Mod", Name: "Example"}},
	}); err != nil {
		t.Fatal(err)
	}
	ids, err := CapturedIDs(dir)
	if err != nil || len(ids) != 1 || ids[0] != "Example.Mod" {
		t.Fatalf("ids %v %v", ids, err)
	}
	edit := Edit{Page: "main", Index: 0, Kind: "int", FieldID: field, Name: "Volume", Value: 8}
	if err := WritePending(dir, "Example.Mod", []Edit{edit}); err != nil {
		t.Fatal(err)
	}
	pending, err := ReadPending(dir, "Example.Mod")
	if err != nil || len(pending.Edits) != 1 {
		t.Fatalf("pending %v %v", pending, err)
	}
	if pending.Edits[0].Value != float64(8) && pending.Edits[0].Value != 8 {
		t.Fatalf("value %v", pending.Edits[0].Value)
	}
	if err := datadir.WriteJSON(ResultPath(dir, "Example.Mod"), Result{
		Schema:  Schema,
		Applied: 0,
		Skipped: []Skipped{{Edit: edit, Reason: "not found"}},
	}); err != nil {
		t.Fatal(err)
	}
	res, err := ReadResult(dir, "Example.Mod")
	if err != nil || len(res.Skipped) != 1 || res.Skipped[0].Reason != "not found" {
		t.Fatalf("result %v %v", res, err)
	}
	if _, err := ReadCapture(dir, "Missing.Mod"); err == nil {
		t.Fatal("expected missing capture")
	}
}

func TestReadPendingRejectsUnknownSchema(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, pendingDir), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteJSON(PendingPath(dir, "Example.Mod"), Pending{
		Schema: Schema + 1,
		Edits:  []Edit{{Name: "Volume"}},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := ReadPending(dir, "Example.Mod")
	if err == nil {
		t.Fatal("expected schema error")
	}
}

func TestParseSetFlag(t *testing.T) {
	t.Parallel()
	page, index, value, err := ParseSetFlag("main/2=true")
	if err != nil || page != "main" || index != 2 || value != "true" {
		t.Fatalf("%s %d %s %v", page, index, value, err)
	}
	if _, _, _, err := ParseSetFlag("nope"); err == nil {
		t.Fatal("expected parse error")
	}
	if _, _, _, err := ParseSetFlag("main/x=1"); err == nil {
		t.Fatal("expected index error")
	}
}

func TestEditFromCapture(t *testing.T) {
	t.Parallel()
	field := "On"
	menu := Capture{Pages: []Page{{ID: "main", Options: []Option{{
		Index: 1, Kind: "bool", FieldID: &field, Name: "On", Value: false,
	}}}}}
	edit, err := EditFromCapture(menu, "main", 1, "true")
	if err != nil {
		t.Fatal(err)
	}
	if edit.Kind != "bool" || edit.Value != true || edit.FieldID != field {
		t.Fatalf("%+v", edit)
	}
}
