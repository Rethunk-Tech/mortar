package contentpatcher_test

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// realProfile reads the profile MORTAR_CP_REAL names as game/id from the data folder HOME points at, which should be a
// sandbox copy: the checks write their caches there and the edit tests change a pack in place, then put it back.
func realProfile(t *testing.T) []framework.Mod {
	t.Helper()
	game, id, ok := strings.Cut(os.Getenv("MORTAR_CP_REAL"), "/")
	if !ok {
		t.Skip("set MORTAR_CP_REAL=game/profile-id, with HOME at a sandbox copy of real data")
	}
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := profiles.Installed(game, id)
	if err != nil {
		t.Fatal(err)
	}
	mods := make([]framework.Mod, len(installed))
	for i, m := range installed {
		mods[i] = framework.Mod{Key: m.Key, Enabled: m.Enabled, Folder: m.Folder, LoadAfter: m.LoadAfter, Manifest: m.Manifest}
	}
	return mods
}

func inputOf(mods []framework.Mod) framework.Input {
	enabled := slices.DeleteFunc(slices.Clone(mods), func(m framework.Mod) bool { return !m.Enabled })
	return framework.Input{Enabled: enabled, All: mods}
}

// fullCheck is a check with nothing kept from an earlier one, in memory or on disk.
func fullCheck(t *testing.T, mods []framework.Mod) framework.Findings {
	t.Helper()
	forgetAll(t)
	return contentpatcher.Driver{}.Analyze(inputOf(mods))
}

func forgetAll(t *testing.T) {
	t.Helper()
	contentpatcher.Driver{}.Forget()
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"problems-content-patcher", "problems-content-patcher-parts.json.gz"} {
		if err := os.RemoveAll(filepath.Join(dir, "cache", name)); err != nil {
			t.Fatal(err)
		}
	}
}

func timedCheck(t *testing.T, label string, mods []framework.Mod) framework.Findings {
	t.Helper()
	start := time.Now()
	got := contentpatcher.Driver{}.Analyze(inputOf(mods))
	t.Logf("%s: %v", label, time.Since(start))
	return got
}

// largestPack is the enabled pack with the biggest content.json, the edit that touches the most targets.
func largestPack(t *testing.T, mods []framework.Mod) int {
	t.Helper()
	best, size := -1, int64(-1)
	for i, m := range mods {
		if !m.Enabled || !strings.EqualFold(m.ContentPackFor, "Pathoschild.ContentPatcher") {
			continue
		}
		if info, err := os.Stat(filepath.Join(m.Folder, "content.json")); err == nil && info.Size() > size {
			best, size = i, info.Size()
		}
	}
	if best < 0 {
		t.Fatal("no enabled Content Patcher pack")
	}
	return best
}

// editPack appends a blank line to the pack's content.json, dated past the window in which a fresh stamp is not
// trusted, and puts the file back after the test.
func editPack(t *testing.T, folder string) {
	t.Helper()
	path := filepath.Join(folder, "content.json")
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = fsx.WriteFile(path, raw, info.Mode())
		_ = os.Chtimes(path, info.ModTime(), info.ModTime())
	})
	if err := fsx.WriteFile(path, append(slices.Clone(raw), '\n'), info.Mode()); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-time.Minute)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func requireSame(t *testing.T, got, want framework.Findings) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("incremental check differs from a full one:\n got %d conflicts, %d settings, %d cleanup, %d redundant\nwant %d conflicts, %d settings, %d cleanup, %d redundant",
			len(got.AssetConflicts), len(got.Settings), len(got.Cleanup), len(got.Redundant),
			len(want.AssetConflicts), len(want.Settings), len(want.Cleanup), len(want.Redundant))
	}
}

func TestRealIncrementalAfterPackEdit(t *testing.T) {
	mods := realProfile(t)
	fullCheck(t, mods)
	timedCheck(t, "unchanged", mods)
	editPack(t, mods[largestPack(t, mods)].Folder)
	got := timedCheck(t, "one pack edited", mods)
	requireSame(t, got, fullCheck(t, mods))
}

func TestRealIncrementalAfterPackAdded(t *testing.T) {
	mods := realProfile(t)
	i := largestPack(t, mods)
	without := slices.Clone(mods)
	without[i].Enabled = false
	fullCheck(t, without)
	got := timedCheck(t, "pack added", mods)
	requireSame(t, got, fullCheck(t, mods))
}

func TestRealIncrementalAfterPackRemoved(t *testing.T) {
	mods := realProfile(t)
	i := largestPack(t, mods)
	fullCheck(t, mods)
	without := slices.Delete(slices.Clone(mods), i, i+1)
	got := timedCheck(t, "pack removed", without)
	requireSame(t, got, fullCheck(t, without))
}
