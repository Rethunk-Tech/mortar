package launch

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCapLogKeepsSMAPIStartupAndTail(t *testing.T) {
	log := "[12:00:00 INFO  SMAPI] SMAPI 4.5.2 with Stardew Valley 1.6.15\n" +
		"[12:00:01 INFO  SMAPI] Loaded 2 mods\n" +
		strings.Repeat("old tail line\n", 30) +
		"newest tail line\n"
	got := CapLog(log, 240)
	if !strings.Contains(got, "Loaded 2 mods") || !strings.Contains(got, "newest tail line") ||
		!strings.Contains(got, omittedStartupLog) {
		t.Fatalf("startup or tail was lost: %q", got)
	}
}

func TestIsCrashIgnoresUpdateAlertsAndModEntryErrors(t *testing.T) {
	cases := []struct {
		entry Entry
		want  bool
	}{
		{Entry{Level: Alert, Mod: "SMAPI", Message: "You can update 8 mods:"}, false},
		{Entry{Level: Error, Mod: "Quest Helper", Message: "Mod crashed on entry and might not work correctly. Technical details:"}, false},
		{Entry{Level: Error, Mod: "SMAPI", Message: "The game failed to launch: NullReferenceException"}, true},
		{Entry{Level: Error, Mod: "SMAPI", Message: "SMAPI failed to initialize: boom"}, true},
		{Entry{Level: Info, Mod: "SMAPI", Message: "fatal"}, false},
	}
	for _, c := range cases {
		if got := IsCrash(c.entry); got != c.want {
			t.Errorf("IsCrash(%q) = %v, want %v", c.entry.Message, got, c.want)
		}
	}
}

func TestSummarizeReadsTheBepInExVersion(t *testing.T) {
	log := "[Message:   BepInEx] BepInEx 5.4.23.5 - Lethal Company (10/4/2026 11:14:16 PM)\n[Error  :  ShipLoot] boom\n"
	if s := Summarize(log); s.SMAPI != "5.4.23.5" || s.Game != "" || s.Errors != 1 {
		t.Fatalf("got %+v", s)
	}
}
