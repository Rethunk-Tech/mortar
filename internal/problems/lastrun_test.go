package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func installedMod(key, name, id string, enabled bool) framework.Mod {
	m := framework.Mod{Key: key, Enabled: enabled}
	m.Name, m.UniqueID = name, id
	return m
}

func TestRunErrorsFromSummary(t *testing.T) {
	mods := []framework.Mod{
		installedMod("a", "Alpha", "author.alpha", true),
		installedMod("b", "Beta", "author.beta", false),
	}
	summary := launch.Summary{
		Crashed: true,
		Mods: []launch.ModError{
			{Mod: "Alpha", Count: 2, First: "first error"},
			{Mod: "Beta", Count: 1, First: "off"},
			{Mod: "Unknown", Count: 1, First: "skip"},
		},
	}
	got := RunErrorsFromSummary("run-1", summary, mods, nil)
	if len(got) != 1 {
		t.Fatalf("got %d problems, want 1: %#v", len(got), got)
	}
	row := got[0]
	if row.Key != "a" || row.ID != "smapi:author.alpha" || row.Count != 2 || row.First != "first error" {
		t.Fatalf("row = %#v", row)
	}
	if !row.Severe || row.RunID != "run-1" {
		t.Fatalf("severe/run = %v %q", row.Severe, row.RunID)
	}
}

func TestRunErrorsFromSummary_cleanRun(t *testing.T) {
	mods := []framework.Mod{installedMod("a", "Alpha", "author.alpha", true)}
	got := RunErrorsFromSummary("run-2", launch.Summary{}, mods, nil)
	if len(got) != 0 {
		t.Fatalf("clean run: %#v", got)
	}
}

func TestRunErrorsFromSummary_infoWhenNoCrash(t *testing.T) {
	mods := []framework.Mod{installedMod("a", "Alpha", "author.alpha", true)}
	summary := launch.Summary{
		Mods: []launch.ModError{{Mod: "author.alpha", Count: 1, First: "oops"}},
	}
	got := RunErrorsFromSummary("run-3", summary, mods, nil)
	if len(got) != 1 || got[0].Severe {
		t.Fatalf("want info-level: %#v", got)
	}
}

func TestRunErrorsFromSummary_matchByUniqueID(t *testing.T) {
	mods := []framework.Mod{installedMod("k", "Display Name", "me.mod", true)}
	summary := launch.Summary{
		Mods: []launch.ModError{{Mod: "me.mod", Count: 1, First: "x"}},
	}
	got := RunErrorsFromSummary("r", summary, mods, nil)
	if len(got) != 1 || got[0].Key != "k" {
		t.Fatalf("match by id: %#v", got)
	}
}

func TestRunErrorsFromSummaryMarksModsUpdatedSinceRun(t *testing.T) {
	im := installedMod("new-key", "Alpha", "me.mod", true)
	im.Version, im.SourceVersion = "2.0", "2.0"
	summary := launch.Summary{
		ModRefs: []launch.ModRef{{
			Key: "old-key", Name: "Alpha", ID: "smapi:me.mod", Version: "1.0", SourceVersion: "1.0",
		}},
		Mods: []launch.ModError{{Mod: "Alpha", Count: 1, First: "old error"}},
	}
	got := RunErrorsFromSummary("run", summary, []framework.Mod{im}, nil)
	if len(got) != 1 || got[0].Key != "new-key" || !got[0].Updated {
		t.Fatalf("updated row = %#v", got)
	}
}

type lastRuns map[string]string

func (r lastRuns) LastRunID(_, profileID string) (string, error) { return r[profileID], nil }

func (lastRuns) LastRunSummary(string, string) (string, launch.Summary, error) {
	return "", launch.Summary{}, nil
}

func TestThePlayerLogBelongsOnlyToTheProfileThatRanLast(t *testing.T) {
	runs := lastRuns{"pack": "20261006T082718-1791275238067000000", "small": "20261006T082334-1791275014043000000"}
	if !ranLast(runs, "lethal-company", "pack", []string{"small", "never"}) {
		t.Fatal("the newest run's profile owns the player log")
	}
	if ranLast(runs, "lethal-company", "small", []string{"pack"}) {
		t.Fatal("another profile ran after this one, so the player log is that profile's")
	}
	if ranLast(runs, "lethal-company", "never", []string{"pack"}) {
		t.Fatal("a profile that never ran owns no player log")
	}
}

func TestRunErrorsFromSummaryNameTheBepInExPackageBehindAPlugin(t *testing.T) {
	pkg := installedMod("k", "SoundAPI", "", true)
	summary := launch.Summary{Mods: []launch.ModError{{Mod: "me.loaforc.soundapi", Count: 3, First: "boom"}, {Mod: "Stranger", Count: 1}}}
	asked := 0
	owners := func() map[string]framework.Mod {
		asked++
		return map[string]framework.Mod{"me.loaforc.soundapi": pkg}
	}
	got := RunErrorsFromSummary("run", summary, []framework.Mod{pkg}, owners)
	if len(got) != 1 || got[0].Key != "k" || got[0].Name != "SoundAPI" || got[0].Count != 3 {
		t.Fatalf("got %+v", got)
	}
	if RunErrorsFromSummary("run", launch.Summary{Mods: []launch.ModError{{Mod: "SoundAPI", Count: 1}}}, []framework.Mod{pkg}, owners); asked != 2 {
		t.Fatalf("owners read %d times; a column naming the mod needs no DLL scan", asked)
	}
}
