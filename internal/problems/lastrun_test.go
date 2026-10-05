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
	got := RunErrorsFromSummary("run-1", summary, mods)
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
	got := RunErrorsFromSummary("run-2", launch.Summary{}, mods)
	if len(got) != 0 {
		t.Fatalf("clean run: %#v", got)
	}
}

func TestRunErrorsFromSummary_infoWhenNoCrash(t *testing.T) {
	mods := []framework.Mod{installedMod("a", "Alpha", "author.alpha", true)}
	summary := launch.Summary{
		Mods: []launch.ModError{{Mod: "author.alpha", Count: 1, First: "oops"}},
	}
	got := RunErrorsFromSummary("run-3", summary, mods)
	if len(got) != 1 || got[0].Severe {
		t.Fatalf("want info-level: %#v", got)
	}
}

func TestRunErrorsFromSummary_matchByUniqueID(t *testing.T) {
	mods := []framework.Mod{installedMod("k", "Display Name", "me.mod", true)}
	summary := launch.Summary{
		Mods: []launch.ModError{{Mod: "me.mod", Count: 1, First: "x"}},
	}
	got := RunErrorsFromSummary("r", summary, mods)
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
	got := RunErrorsFromSummary("run", summary, []framework.Mod{im})
	if len(got) != 1 || got[0].Key != "new-key" || !got[0].Updated {
		t.Fatalf("updated row = %#v", got)
	}
}
