package gmcm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	if err := datadir.WriteJSON(CapturePath(dir, "smapi:Example.Mod"), menu); err != nil {
		t.Fatal(err)
	}
	got, err := ReadCapture(dir, "smapi:Example.Mod")
	if err != nil {
		t.Fatal(err)
	}
	if got.Mod.ID != "Example.Mod" || len(got.Pages) != 1 || got.Pages[0].Options[0].Name != "Volume" {
		t.Fatalf("capture: %+v", got)
	}
	edit := Edit{Page: "main", Index: 0, Kind: "int", FieldID: field, Name: "Volume", Value: 8}
	if err := WritePending(dir, "smapi:Example.Mod", []Edit{edit}); err != nil {
		t.Fatal(err)
	}
	pending, err := ReadPending(dir, "smapi:Example.Mod")
	if err != nil || len(pending.Edits) != 1 {
		t.Fatalf("pending %v %v", pending, err)
	}
	if pending.Edits[0].Value != float64(8) && pending.Edits[0].Value != 8 {
		t.Fatalf("value %v", pending.Edits[0].Value)
	}
	if err := datadir.WriteJSON(ResultPath(dir, "smapi:Example.Mod"), Result{
		Schema:  Schema,
		Skipped: []Skipped{{Edit: edit, Reason: "not found"}},
	}); err != nil {
		t.Fatal(err)
	}
	res, err := ReadResult(dir, "smapi:Example.Mod")
	if err != nil || len(res.Skipped) != 1 || res.Skipped[0].Reason != "not found" {
		t.Fatalf("result %v %v", res, err)
	}
	if _, err := ReadCapture(dir, "smapi:Missing.Mod"); err == nil {
		t.Fatal("expected missing capture")
	}
}

func TestReadPendingRejectsUnknownSchema(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, pendingDir), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteJSON(PendingPath(dir, "smapi:Example.Mod"), Pending{
		Schema: Schema + 1,
		Edits:  []Edit{{Name: "Volume"}},
	}); err != nil {
		t.Fatal(err)
	}
	_, err := ReadPending(dir, "smapi:Example.Mod")
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
		Index: 1, Kind: "bool", FieldID: &field, Name: "On", Value: false, Editable: true,
	}}}}}
	edit, err := EditFromCapture(menu, "main", 1, "true")
	if err != nil {
		t.Fatal(err)
	}
	if edit.Kind != "bool" || edit.Value != true || edit.FieldID != field {
		t.Fatalf("%+v", edit)
	}
}

// The fixtures are the files mortar-smapi-bridge's own tests write (GmcmWalkerTests), copied byte for byte.
func TestReadsTheBridgeFixtures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for name, dest := range map[string]string{
		"gmcm-capture.json": CapturePath(dir, "smapi:demo.Mod"),
		"gmcm-result.json":  ResultPath(dir, "smapi:demo.Mod"),
	} {
		raw, err := fsx.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := datadir.WriteFile(dest, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	menu, err := ReadCapture(dir, "smapi:demo.Mod")
	if err != nil {
		t.Fatal(err)
	}
	count, err := Find(menu, "", 2)
	if err != nil || count.Kind != "int" || *count.Min != 1 || *count.Max != 5 || len(count.FormatSamples) != 3 || count.FormatSamples[0].Label != "n=1" {
		t.Fatalf("count = %+v, %v", count, err)
	}
	pick, _ := Find(menu, "", 3)
	if pick.Kind != "choice" || len(pick.Choices) == 0 || Text(pick.Value) == "" {
		t.Fatalf("pick = %+v", pick)
	}
	image, _ := Find(menu, "", 8)
	if image.Kind != "image" || Text(image.Value) != "1" {
		t.Fatalf("image = %+v", image)
	}
	result, err := ReadResult(dir, "smapi:demo.Mod")
	if err != nil || len(result.Applied) != 1 || result.Applied[0].Name != "Count" || len(result.Skipped) != 1 || result.Skipped[0].Reason != "kind/name mismatch" {
		t.Fatalf("result = %+v, %v", result, err)
	}

	if _, err := EditFromCapture(menu, "", 2, "9"); err == nil {
		t.Fatal("a value above the maximum was accepted")
	}
	if _, err := EditFromCapture(menu, "", 3, "not-a-choice"); err == nil {
		t.Fatal("a value the choice does not offer was accepted")
	}
	if _, err := EditFromCapture(menu, "", 6, "x"); err == nil {
		t.Fatal("an option only the game can change was accepted")
	}
	edit, err := EditFromCapture(menu, "", 2, "5")
	if err != nil || edit.Value != 5 || edit.FieldID != "count" {
		t.Fatalf("edit = %+v, %v", edit, err)
	}
}
