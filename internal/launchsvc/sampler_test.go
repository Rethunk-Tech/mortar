package launchsvc

import (
	"maps"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-nettrace"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAssembliesToModsSkipsUnreadableManifest(t *testing.T) {
	t.Parallel()
	mods := t.TempDir()
	testfs.WriteAt(t, filepath.Join(mods, "A", "manifest.json"), []byte(`{"Name":"A","UniqueID":"a.b","Version":"1.0.0"}`))
	testfs.WriteAt(t, filepath.Join(mods, "A", "A.dll"), []byte("x"))
	testfs.WriteAt(t, filepath.Join(mods, "B", "manifest.json"), []byte("not json"))
	testfs.WriteAt(t, filepath.Join(mods, "B", "B.dll"), []byte("x"))
	got, err := assembliesToMods(mods)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"A": "a.b"}; !maps.Equal(got, want) {
		t.Fatalf("assembliesToMods = %v, want %v", got, want)
	}
}

func TestMainSampleThreadPicksThreadWithModFrames(t *testing.T) {
	t.Parallel()
	mods := map[string]string{"A": "a.b"}
	modFrame := []nettrace.Frame{{Assembly: "A, Version=1.0.0.0"}}
	other := []nettrace.Frame{{Assembly: "Stardew Valley"}}
	samples := []nettrace.Sample{{ThreadID: 3, Frames: other}, {ThreadID: 7, Frames: modFrame}, {ThreadID: 3, Frames: other}}
	if id, ok := mainSampleThread(samples, mods); id != 7 || !ok {
		t.Fatalf("mainSampleThread = %d, %v, want 7, true", id, ok)
	}
	if _, ok := mainSampleThread(samples[:1], mods); ok {
		t.Fatal("a thread without mod frames was picked")
	}
}
