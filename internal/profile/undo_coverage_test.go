package profile

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// undoFixture is a profile with two mods (Me.A carries a config.json), a group, launch settings and a preset.
func undoFixture(t *testing.T) (env, Profile) {
	t.Helper()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A"), "config.json": `{"v":1}`})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	e.item(t, "local-c", map[string]string{"manifest.json": manifestJSON("Me.C")})
	e.item(t, "local-a2", map[string]string{"manifest.json": `{"Name":"Me.A","Author":"me","Version":"2.0.0","UniqueID":"Me.A"}`})
	p := mustCreate(t, e, "Farm")
	for _, k := range []string{"local-a", "local-b"} {
		if _, err := e.AddEntry("stardew", p.ID, k, Source{Kind: KindLocal, Name: k + ".zip"}); err != nil {
			t.Fatal(err)
		}
	}
	steps := []func() (Profile, error){
		func() (Profile, error) { return e.CreateGroup("stardew", p.ID, "G") },
		func() (Profile, error) { return e.AddToGroup("stardew", p.ID, "G", "local-a") },
		func() (Profile, error) { return e.SetLaunchOptions("stardew", p.ID, "--x") },
		func() (Profile, error) {
			return e.SetCollection("stardew", p.ID, CollectionRef{Domain: "stardewvalley", Slug: "c", Name: "C", Revision: 1})
		},
		func() (Profile, error) {
			return e.AddLaunchPreset("stardew", p.ID, LaunchPreset{Name: "Fast", LaunchOptions: "--fast"})
		},
	}
	for _, step := range steps {
		if _, err := step(); err != nil {
			t.Fatal(err)
		}
	}
	return e, p
}

func sameState(a, b Profile) bool {
	x, _ := json.Marshal(stateOf(a))
	y, _ := json.Marshal(stateOf(b))
	return bytes.Equal(x, y)
}

func liveConfig(e env, p Profile) string {
	b, err := fsx.ReadFile(e.mods(p.ID) + "/local-a/config.json")
	if err != nil {
		return "<absent>"
	}
	return string(b)
}

// TestEveryProfileChangeRecordsAnEventAndUndoesExactly runs each mutation of a profile's mods, order, groups, launch
// settings and config, then reverts to the event recorded before it and requires the profile to be as it was.
func TestEveryProfileChangeRecordsAnEventAndUndoesExactly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		act  func(e env, p Profile) error
	}{
		{"AddEntry", func(e env, p Profile) error {
			_, err := e.AddEntry("stardew", p.ID, "local-c", Source{Kind: KindLocal, Name: "c.zip"})
			return err
		}},
		{"RemoveEntry", func(e env, p Profile) error { _, err := e.RemoveEntry("stardew", p.ID, "local-b"); return err }},
		{"RemoveEntries", func(e env, p Profile) error {
			_, err := e.RemoveEntries("stardew", p.ID, []string{"local-a", "local-b"})
			return err
		}},
		{"SetModEnabled", func(e env, p Profile) error {
			_, err := e.SetModEnabled("stardew", p.ID, "local-b", "smapi:Me.B", false)
			return err
		}},
		{"SetModsEnabled", func(e env, p Profile) error {
			_, err := e.SetModsEnabled("stardew", p.ID, []EnableRef{{Key: "local-a", ID: "smapi:Me.A"}, {Key: "local-b", ID: "smapi:Me.B"}}, false)
			return err
		}},
		{"SetPinned", func(e env, p Profile) error {
			_, err := e.SetPinned("stardew", p.ID, "local-a", true, "why")
			return err
		}},
		{"SetPinnedMany", func(e env, p Profile) error {
			_, err := e.SetPinnedMany("stardew", p.ID, []string{"local-a", "local-b"}, true, "")
			return err
		}},
		{"SetSkipVersion", func(e env, p Profile) error {
			_, err := e.SetSkipVersion("stardew", p.ID, "local-a", "9.9.9")
			return err
		}},
		{"SetSkipSource", func(e env, p Profile) error {
			_, err := e.SetSkipSource("stardew", p.ID, "local-a", "Nexus", true)
			return err
		}},
		{"SetEntryNoteTags", func(e env, p Profile) error {
			_, err := e.SetEntryNoteTags("stardew", p.ID, "local-a", "a note", []string{"x", "y"})
			return err
		}},
		{"SetEntryCategory", func(e env, p Profile) error {
			_, err := e.SetEntryCategory("stardew", p.ID, "local-a", "Crops")
			return err
		}},
		{"SetEntryCategoryMany", func(e env, p Profile) error {
			_, err := e.SetEntryCategoryMany("stardew", p.ID, []string{"local-a", "local-b"}, "Crops")
			return err
		}},
		{"SetEntryTagsMany", func(e env, p Profile) error {
			_, err := e.SetEntryTagsMany("stardew", p.ID, []string{"local-a", "local-b"}, "t", true)
			return err
		}},
		{"UpdateEntry", func(e env, p Profile) error {
			_, err := e.UpdateEntry("stardew", p.ID, "local-a", "local-a2")
			return err
		}},
		{"SetWinner", func(e env, p Profile) error {
			_, err := e.SetWinner("stardew", p.ID, "local-a", "smapi:Me.A", mod.ID("smapi:Me.B"), true)
			return err
		}},
		{"FollowOrder (load order)", func(e env, p Profile) error {
			return e.FollowOrder("stardew", p.ID, func(en Entry) (int, bool) {
				if en.Key == "local-b" {
					return 0, true
				}
				return 1, true
			})
		}},
		{"PlaceArrivals (load order)", func(e env, p Profile) error {
			_, err := e.PlaceArrivals("stardew", p.ID, func(en Entry) (int, bool) {
				if en.Key == "local-b" {
					return 0, true
				}
				return 1, true
			}, []string{"local-a"})
			return err
		}},
		{"CreateGroup", func(e env, p Profile) error { _, err := e.CreateGroup("stardew", p.ID, "H"); return err }},
		{"RenameGroup", func(e env, p Profile) error { _, err := e.RenameGroup("stardew", p.ID, "G", "G2"); return err }},
		{"DeleteGroup", func(e env, p Profile) error { _, err := e.DeleteGroup("stardew", p.ID, "G"); return err }},
		{"AddToGroup", func(e env, p Profile) error { _, err := e.AddToGroup("stardew", p.ID, "G", "local-b"); return err }},
		{"RemoveFromGroup", func(e env, p Profile) error { _, err := e.RemoveFromGroup("stardew", p.ID, "G", "local-a"); return err }},
		{"SetGroupEnabled", func(e env, p Profile) error { _, err := e.SetGroupEnabled("stardew", p.ID, "G", false); return err }},
		{"SetLaunchOptions", func(e env, p Profile) error { _, err := e.SetLaunchOptions("stardew", p.ID, "--y"); return err }},
		{"SetLaunchSettings", func(e env, p Profile) error {
			_, err := e.SetLaunchSettings("stardew", p.ID, "gamemoderun", "A=1")
			return err
		}},
		{"AddLaunchPreset", func(e env, p Profile) error {
			_, err := e.AddLaunchPreset("stardew", p.ID, LaunchPreset{Name: "Slow"})
			return err
		}},
		{"SetLaunchPresets", func(e env, p Profile) error { _, err := e.SetLaunchPresets("stardew", p.ID, nil, ""); return err }},
		{"SetDefaultLaunchPreset", func(e env, p Profile) error {
			cur, err := e.read("stardew", p.ID)
			if err != nil {
				return err
			}
			_, err = e.SetDefaultLaunchPreset("stardew", p.ID, cur.LaunchPresets[0].ID)
			return err
		}},
		{"SetLoader", func(e env, p Profile) error { _, err := e.SetLoader("stardew", p.ID, "smapi"); return err }},
		{"SetInstall", func(e env, p Profile) error { _, err := e.SetInstall("stardew", p.ID, "abc"); return err }},
		{"SetSeparateSaves", func(e env, p Profile) error { _, err := e.SetSeparateSaves("stardew", p.ID, true, false); return err }},
		{"SetOverride", func(e env, p Profile) error {
			_, err := e.SetOverride("stardew", p.ID, overrideSkipPlayCheck, "true")
			return err
		}},
		{"ClearCollection", func(e env, p Profile) error { _, err := e.ClearCollection("stardew", p.ID); return err }},
		{"WriteConfig", func(e env, p Profile) error {
			return e.WriteConfig("stardew", p.ID, "local-a", "smapi:Me.A", `{"v":2}`)
		}},
		{"ResetConfig", func(e env, p Profile) error { return e.ResetConfig("stardew", p.ID, "local-a", "smapi:Me.A") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e, p := undoFixture(t)
			before, err := e.read("stardew", p.ID)
			if err != nil {
				t.Fatal(err)
			}
			cfgBefore := liveConfig(e, p)
			baseline, err := e.Baseline("stardew", p.ID)
			if err != nil {
				t.Fatal(err)
			}
			eventsBefore, err := e.History("stardew", p.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.act(e, p); err != nil {
				t.Fatal(err)
			}
			eventsAfter, err := e.History("stardew", p.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(eventsAfter) <= len(eventsBefore) {
				t.Fatalf("%s recorded no history event", tc.name)
			}
			after, err := e.read("stardew", p.ID)
			if err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(after.Entries, before.Entries) && sameState(after, before) &&
				liveConfig(e, p) == cfgBefore {
				t.Fatalf("%s changed nothing the test can see", tc.name)
			}
			got, err := e.Revert("stardew", p.ID, baseline)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Entries, before.Entries) {
				t.Errorf("entries after undo = %+v, want %+v", got.Entries, before.Entries)
			}
			if !sameState(got, before) {
				t.Errorf("settings after undo = %+v, want %+v", stateOf(got), stateOf(before))
			}
			if cfg := liveConfig(e, p); cfg != cfgBefore {
				t.Errorf("config after undo = %q, want %q", cfg, cfgBefore)
			}
		})
	}
}

// Deleting a custom category clears the override on every entry that used it; each profile it touched records that.
func TestDeletingACustomCategoryRecordsAnEventInEachProfileItChanged(t *testing.T) {
	t.Parallel()
	e, p := undoFixture(t)
	saved, err := e.SaveCustomCategories("stardew", []CustomCategory{{Name: "QoL", Color: "teal"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetEntryCategory("stardew", p.ID, "local-a", saved[0].ID); err != nil {
		t.Fatal(err)
	}
	before, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SaveCustomCategories("stardew", nil); err != nil {
		t.Fatal(err)
	}
	after, err := e.History("stardew", p.ID)
	if err != nil || len(after) != len(before)+1 {
		t.Fatalf("events %d -> %d, %v", len(before), len(after), err)
	}
	prev, err := e.Revert("stardew", p.ID, after[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if i := entryIndex(prev.Entries, "local-a"); i < 0 || prev.Entries[i].CategoryOverride != saved[0].ID {
		t.Fatalf("undo left %+v", prev.Entries)
	}
}

// A history written before events carried settings has no state; reverting to one of its events restores the entries
// and leaves every setting as it is now.
func TestRevertToAnEventWithoutStateKeepsCurrentSettings(t *testing.T) {
	t.Parallel()
	e, p := undoFixture(t)
	baseline, err := e.Baseline("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, historyFile)
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		FormatVersion int                          `json:"formatVersion"`
		Counted       bool                         `json:"counted"`
		Events        []map[string]json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, ev := range doc.Events {
		delete(ev, "state")
	}
	raw, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "local-c", Source{Kind: KindLocal, Name: "c.zip"}); err != nil {
		t.Fatal(err)
	}
	for _, step := range []func() (Profile, error){
		func() (Profile, error) { return e.SetLoader("stardew", p.ID, "smapi") },
		func() (Profile, error) { return e.SetInstall("stardew", p.ID, "abc") },
		func() (Profile, error) { return e.SetSeparateSaves("stardew", p.ID, true, false) },
		func() (Profile, error) { return e.SetOverride("stardew", p.ID, overrideSkipPlayCheck, "true") },
		func() (Profile, error) { return e.SetLaunchOptions("stardew", p.ID, "--now") },
		func() (Profile, error) { return e.DeleteGroup("stardew", p.ID, "G") },
	} {
		if _, err := step(); err != nil {
			t.Fatal(err)
		}
	}
	current, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Revert("stardew", p.ID, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("entries after revert = %d, want the 2 before local-c", len(got.Entries))
	}
	if !sameState(got, current) {
		t.Fatalf("settings after revert = %+v, want the current %+v", stateOf(got), stateOf(current))
	}
}

// undoesExactly runs act on a profile, requires an event and an actual change, then reverts to the baseline taken
// before it and requires the entries and settings to be as they were.
func undoesExactly(t *testing.T, s *Store, game string, p Profile, act func() error) {
	t.Helper()
	before, err := s.read(game, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Baseline(game, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore, err := s.History(game, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := act(); err != nil {
		t.Fatal(err)
	}
	eventsAfter, err := s.History(game, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventsAfter) <= len(eventsBefore) {
		t.Fatal("the change recorded no history event")
	}
	after, err := s.read(game, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(after.Entries, before.Entries) && sameState(after, before) {
		t.Fatal("the change altered nothing")
	}
	got, err := s.Revert(game, p.ID, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Entries, before.Entries) {
		t.Errorf("entries after undo = %+v, want %+v", got.Entries, before.Entries)
	}
	if !sameState(got, before) {
		t.Errorf("settings after undo = %+v, want %+v", stateOf(got), stateOf(before))
	}
}

// splitFixture installs a Nexus main file with one extra file combined into it.
func splitFixture(t *testing.T) (env, Profile, string, string) {
	t.Helper()
	e := newEnv(t)
	p := mustCreate(t, e, "P")
	main := buildZip(t, "a.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	res, err := e.InstallSource("stardew", p.ID, main, Source{Kind: KindNexus, Name: "a.zip", ModID: 7, FileID: 1, Version: "1.0"})
	if err != nil {
		t.Fatal(err)
	}
	entryKey, extraKey := res.Profile.Entries[0].Key, store.NexusKey(7, 2)
	e.item(t, extraKey, map[string]string{"B/manifest.json": manifestJSON("X.B")})
	if _, err := e.AddExtra("stardew", p.ID, entryKey, extraKey, Source{Kind: KindNexus, ModID: 7, FileID: 2}); err != nil {
		t.Fatal(err)
	}
	return e, p, entryKey, extraKey
}

func TestPackageBackedChangesUndoExactly(t *testing.T) {
	t.Parallel()
	t.Run("SplitExtra", func(t *testing.T) {
		t.Parallel()
		e, p, entryKey, extraKey := splitFixture(t)
		undoesExactly(t, e.Store, "stardew", p, func() error { _, err := e.SplitExtra("stardew", p.ID, entryKey, extraKey); return err })
	})
	t.Run("CombineEntries", func(t *testing.T) {
		t.Parallel()
		e, p, entryKey, extraKey := splitFixture(t)
		if _, err := e.SplitExtra("stardew", p.ID, entryKey, extraKey); err != nil {
			t.Fatal(err)
		}
		undoesExactly(t, e.Store, "stardew", p, func() error { _, err := e.CombineEntries("stardew", p.ID, entryKey, extraKey); return err })
	})
	t.Run("SetOverlayEnabled", func(t *testing.T) {
		t.Parallel()
		e, p, _, optKey := installOverlayPair(t)
		off := overlayEntry(p, optKey).OverlayOff
		undoesExactly(t, e.Store, "stardew", p, func() error { _, err := e.SetOverlayEnabled("stardew", p.ID, optKey, off); return err })
	})
	t.Run("SetUpdateChannel", func(t *testing.T) {
		t.Parallel()
		e := newEnv(t)
		key := store.NexusKey(1, 1)
		e.item(t, key, map[string]string{"manifest.json": manifestJSON("Me.A")})
		p := mustCreate(t, e, "P")
		if _, err := e.AddEntry("stardew", p.ID, key, Source{Kind: KindNexus, Name: "a.zip", ModID: 1, FileID: 1, Version: "1.0.0"}); err != nil {
			t.Fatal(err)
		}
		undoesExactly(t, e.Store, "stardew", p, func() error { _, err := e.SetUpdateChannel("stardew", p.ID, key, "optional"); return err })
	})
	t.Run("MovePackage", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		s := OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
		const lc = "lethal-company"
		p := mustCreateIn(t, s, lc, "Friends")
		var keys []string
		for _, name := range []string{"A", "B"} {
			zip := buildZip(t, name+".zip", map[string]string{
				"manifest.json":     `{"name":"` + name + `","version_number":"1.0.0"}`,
				"config/Shared.cfg": name,
			})
			res, err := s.InstallSource(lc, p.ID, zip, Source{Kind: KindThunderstore, Name: "Ns-" + name, Version: "1.0.0"})
			if err != nil {
				t.Fatal(err)
			}
			keys = append(keys, res.Profile.Entries[len(res.Profile.Entries)-1].Key)
		}
		undoesExactly(t, s, lc, p, func() error { _, err := s.MovePackage(lc, p.ID, keys[1], -1); return err })
	})
}

func mustCreateIn(t *testing.T, s *Store, gameID, name string) Profile {
	t.Helper()
	p, err := s.Create(gameID, name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// The Undo on a revert's toast reverts to the event before it; the configs that revert put back come back too.
func TestUndoOfRevertRestoresConfig(t *testing.T) {
	e, p := undoFixture(t)
	baseline, err := e.Baseline("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	orig := liveConfig(e, p)
	if err := e.WriteConfig("stardew", p.ID, "local-a", "smapi:Me.A", `{"v":2}`); err != nil {
		t.Fatal(err)
	}
	edited := liveConfig(e, p)
	evs, err := e.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	head := evs[0].ID
	if _, err := e.Revert("stardew", p.ID, baseline); err != nil {
		t.Fatal(err)
	}
	if got := liveConfig(e, p); got != orig {
		t.Fatalf("revert left config %q, want %q", got, orig)
	}
	if _, err := e.Revert("stardew", p.ID, head); err != nil {
		t.Fatal(err)
	}
	if got := liveConfig(e, p); got != edited {
		t.Fatalf("undo of the revert left config %q, want the edit %q", got, edited)
	}
}
