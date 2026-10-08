package profile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func modJSON(id, version string) map[string]string {
	return map[string]string{"manifest.json": `{"Name":"` + id + `","Author":"me","Version":"` + version + `","UniqueID":"` + id + `"}`}
}

func (e env) addLocal(t *testing.T, profileID, key, id, version string) {
	t.Helper()
	e.item(t, key, modJSON(id, version))
	if _, err := e.AddEntry("stardew", profileID, key, Source{Kind: KindLocal, Name: key + ".zip"}); err != nil {
		t.Fatal(err)
	}
}

// mergeEnv: source has Only (disabled, edited config), Same 1.0, Newer 2.0, Older 1.0; target has Same 1.0, Newer 1.0, Older 2.0, Keep.
func mergeEnv(t *testing.T) (env, Profile, Profile) {
	t.Helper()
	e := newEnv(t)
	src, _ := e.Create("stardew", "Src")
	dst, _ := e.Create("stardew", "Dst")
	e.addLocal(t, src.ID, "only-1", "Me.Only", "1.0.0")
	dir, err := e.ModFolder("stardew", src.ID, "only-1", "smapi:Me.Only")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", src.ID, "only-1", "smapi:Me.Only", false); err != nil {
		t.Fatal(err)
	}
	e.addLocal(t, src.ID, "same-1", "Me.Same", "1.0.0")
	e.addLocal(t, src.ID, "newer-2", "Me.Newer", "2.0.0")
	e.addLocal(t, src.ID, "older-1", "Me.Older", "1.0.0")
	e.addLocal(t, dst.ID, "same-1", "Me.Same", "1.0.0")
	e.addLocal(t, dst.ID, "newer-1", "Me.Newer", "1.0.0")
	e.addLocal(t, dst.ID, "older-2", "Me.Older", "2.0.0")
	e.addLocal(t, dst.ID, "keep-1", "Me.Keep", "1.0.0")
	return e, src, dst
}

func TestMergePreviewBuckets(t *testing.T) {
	t.Parallel()
	e, src, dst := mergeEnv(t)
	pv, err := e.MergePreview("stardew", src.ID, dst.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pv.Adds) != 1 || pv.Adds[0].ID != "smapi:Me.Only" || pv.Adds[0].Enabled || len(pv.Both) != 3 || pv.TargetOnly != 1 {
		t.Fatalf("preview = %+v", pv)
	}
	newer := map[string]bool{}
	for _, b := range pv.Both {
		newer[b.ID.Local()] = b.SourceNewer
	}
	if !reflect.DeepEqual(newer, map[string]bool{"Me.Same": false, "Me.Newer": true, "Me.Older": false}) {
		t.Fatalf("newer = %v", newer)
	}
}

func TestMergeIntoAddsAndKeepsDuplicates(t *testing.T) {
	t.Parallel()
	e, src, dst := mergeEnv(t)
	srcBefore, _ := e.load("stardew", src.ID)
	got, err := e.MergeInto("stardew", src.ID, dst.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]Entry{}
	for _, en := range got.Entries {
		keys[en.Key] = en
	}
	if _, ok := keys["only-1"]; !ok || keys["only-1"].Enabled("smapi:Me.Only") || keys["newer-1"].Key == "" || keys["older-2"].Key == "" || len(keys) != 5 {
		t.Fatalf("entries = %v", keys)
	}
	dir, _ := e.ModFolder("stardew", dst.ID, "only-1", "smapi:Me.Only")
	if b, err := fsx.ReadFile(filepath.Join(dir, "config.json")); err != nil || string(b) != `{"a":1}` {
		t.Fatalf("config = %q, %v", b, err)
	}
	if after, _ := e.load("stardew", src.ID); !reflect.DeepEqual(after.Entries, srcBefore.Entries) {
		t.Fatal("source changed")
	}
}

func TestMergeIntoNewerWinsAndUndo(t *testing.T) {
	t.Parallel()
	e, src, dst := mergeEnv(t)
	before, _ := e.load("stardew", dst.ID)
	eventsBefore, _ := e.History("stardew", dst.ID)
	got, err := e.MergeInto("stardew", src.ID, dst.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, en := range got.Entries {
		keys[en.Key] = true
	}
	if !keys["newer-2"] || keys["newer-1"] || !keys["older-2"] || keys["older-1"] {
		t.Fatalf("keys = %v", keys)
	}
	events, _ := e.History("stardew", dst.ID)
	if len(events) != len(eventsBefore)+1 || events[0].Change != ChangeMerged || events[0].Name != "Src" || events[0].Count != 2 {
		t.Fatalf("events = %+v", events)
	}
	if _, err := e.Revert("stardew", dst.ID, eventsBefore[0].ID); err != nil {
		t.Fatal(err)
	}
	after, _ := e.load("stardew", dst.ID)
	if !reflect.DeepEqual(after.Entries, before.Entries) {
		t.Fatalf("undo: %+v want %+v", after.Entries, before.Entries)
	}
	if _, err := e.ModFolder("stardew", dst.ID, "only-1", "smapi:Me.Only"); err == nil {
		t.Fatal("only-1 still there")
	}
}

func TestMergeIntoRefusedWhileRunning(t *testing.T) {
	t.Parallel()
	e, src, dst := mergeEnv(t)
	e.Running = func(_, id string) bool { return id == dst.ID }
	var re *RunningError
	if _, err := e.MergeInto("stardew", src.ID, dst.ID, true); !errors.As(err, &re) {
		t.Fatalf("err = %v", err)
	}
}
