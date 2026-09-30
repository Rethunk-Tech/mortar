package sharesvc

import (
	"archive/zip"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/share"
)

func modZip(t *testing.T, uniqueID string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mod.zip")
	f, err := fsx.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("Mod/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	manifest := `{"Name":"Mod","Author":"a","Version":"1.0.0","UniqueID":"` + uniqueID + `","EntryDll":"Mod.dll"}`
	if _, err := w.Write([]byte(manifest)); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(zw.Close(), f.Close()); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPendingConfigsSurviveARestart(t *testing.T) {
	first, _ := newService(t, true)
	first.d.Dir = t.TempDir()
	prof, err := first.d.Profiles.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	first.pending = []*pending{{
		Game: "stardew", Profile: prof.ID,
		Wanted:  []wantedFile{{ModID: 100, FileID: 1}},
		Configs: []share.Config{{UniqueID: "A.Mod", Path: "config.json", Data: []byte(`{"a":1}`)}},
	}}
	first.savePending()

	// Mortar quits before the mod finishes installing, and starts again.
	second := NewService(first.d)
	if len(second.pending) != 1 {
		t.Fatalf("pending after restart = %d, want 1", len(second.pending))
	}
	res, err := second.d.Profiles.InstallNexus("stardew", prof.ID, modZip(t, "A.Mod"), profile.Source{Kind: profile.KindNexus, ModID: 100, FileID: 1})
	if err != nil {
		t.Fatal(err)
	}
	done := queue.Item{Game: "stardew", Profile: prof.ID, ModID: 100, FileID: 1, State: queue.StateDone}
	second.QueueChanged(queue.State{Items: []queue.Item{done}})

	dir, err := second.d.Profiles.ModsDir("stardew", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	key := res.Profile.Entries[0].Key
	got, err := fsx.ReadFile(filepath.Join(dir, key, "Mod", "config.json"))
	if err != nil || string(got) != `{"a":1}` {
		t.Fatalf("config = %q, %v", got, err)
	}
	if len(second.pending) != 0 {
		t.Fatalf("settled import still pending: %d", len(second.pending))
	}
	if again := NewService(first.d); len(again.pending) != 0 {
		t.Fatalf("settled import came back after another restart")
	}
}
