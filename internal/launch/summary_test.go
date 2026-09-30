package launch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSummarizeAttributesErrorsFromFixture(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "errors.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := Summarize(string(body))
	if got.SMAPI != "4.5.2" || got.Game != "1.6.15" {
		t.Fatalf("versions = %q / %q", got.SMAPI, got.Game)
	}
	if !got.Crashed {
		t.Fatal("ALERT crash line must mark the run crashed")
	}
	if got.Warnings != 1 {
		t.Fatalf("warnings = %d", got.Warnings)
	}
	// Content Patcher 2, Lookup Anything 1, SMAPI alert; Galaxy's ERROR game lines are suppressed.
	if got.Errors != 4 {
		t.Fatalf("errors = %d", got.Errors)
	}
	if len(got.Mods) != 3 {
		t.Fatalf("mods = %#v", got.Mods)
	}
	if got.Mods[0].Mod != "Content Patcher" || got.Mods[0].Count != 2 || got.Mods[0].First != "patch failed: missing target" {
		t.Fatalf("first mod = %#v", got.Mods[0])
	}
	if got.Mods[1].Mod != "Lookup Anything" || got.Mods[1].Count != 1 || got.Mods[1].First != "null reference" {
		t.Fatalf("second mod = %#v", got.Mods[1])
	}
	if got.Mods[2].Mod != "SMAPI" || got.Mods[2].Count != 1 {
		t.Fatalf("SMAPI = %#v", got.Mods[2])
	}
}

func TestSummarizeOrdinaryErrorsAreNotACrash(t *testing.T) {
	log := "[19:43:50 ERROR SomeMod] Failed to load a content pack\n"
	got := Summarize(log)
	if got.Crashed || got.Errors != 1 || len(got.Mods) != 1 || got.Mods[0].Mod != "SomeMod" {
		t.Fatalf("got %#v", got)
	}
}

func TestCapLogKeepsTheTailOnALineBoundary(t *testing.T) {
	log := "aaaa\nbbbb\ncccc\n"
	if CapLog(log, 10) != "cccc\n" {
		t.Fatalf("got %q", CapLog(log, 10))
	}
	if CapLog("short", 100) != "short" {
		t.Fatal("short logs are unchanged")
	}
}
