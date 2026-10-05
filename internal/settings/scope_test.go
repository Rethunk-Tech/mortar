package settings

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestFileIsSplitByScope(t *testing.T) {
	s, dir := open(t)
	if _, err := s.Update(func(v *Settings) {
		v.Accent = "moss"
		v.NexusName = "Farmer"
		v.NxmPreviousHandlers = map[string]string{"nxm": "vortex.desktop"}
		v.AutoTrackNexus = true
		if err := ApplyKeyGame(v, "runsKept", "7", "stardew"); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Global  map[string]any            `json:"global"`
		Sources map[string]map[string]any `json:"sources"`
		Loaders map[string]map[string]any `json:"loaders"`
		Games   map[string]map[string]any `json:"games"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	nexus := doc.Sources["nexus"]
	if doc.Global["accent"] != "moss" || nexus["name"] != "Farmer" || nexus["autoTrack"] != true ||
		doc.Games["stardew"]["runsKept"] != float64(7) {
		t.Fatalf("scopes: %s", b)
	}
	if _, ok := doc.Global["nexusName"]; ok {
		t.Fatalf("a source key sits in global: %s", b)
	}
	if doc.Loaders["smapi"]["builds"] != "show" || doc.Loaders["smapi"]["tellWhenOut"] != true || doc.Games["stardew"]["smapiBuilds"] != nil {
		t.Fatalf("loader scope: %s", b)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get(); got.NexusName != "Farmer" || got.NxmPreviousHandlers["nxm"] != "vortex.desktop" || got.GamePrefs("stardew").RunsKept != 7 {
		t.Fatalf("reload = %+v", got)
	}
}

func TestResolveAtWalksTheScopeTuple(t *testing.T) {
	s := Defaults()
	if err := ApplyKeyGame(&s, "runsKept", "9", "stardew"); err != nil {
		t.Fatal(err)
	}
	if err := ApplyKeyGame(&s, "verifyNexusMD5", "true", ""); err != nil {
		t.Fatal(err)
	}
	if got := ResolveAt(s, "runsKept", Scope{Game: "stardew"}, nil); got != "9" {
		t.Fatalf("game = %q", got)
	}
	if got := ResolveAt(s, "runsKept", Scope{}, nil); got != PrefDefault(t, "runsKept") {
		t.Fatalf("no game = %q", got)
	}
	if got := ResolveAt(s, "verifyNexusMD5", Scope{Source: "nexus", Game: "stardew"}, nil); got != "true" {
		t.Fatalf("source = %q", got)
	}
	if got := ResolveAt(s, "saveBackupsKept", Scope{Game: "stardew", Profile: "p"}, map[string]string{"saveBackupsKept": "3"}); got != "3" {
		t.Fatalf("profile override = %q", got)
	}
	if got := ResolveAt(s, "runsKept", Scope{Game: "stardew", Profile: "p"}, map[string]string{"runsKept": "1"}); got != "9" {
		t.Fatalf("a key that is not overridable took an override: %q", got)
	}
}

// PrefDefault is the registry default of key.
func PrefDefault(t *testing.T, key string) string {
	t.Helper()
	p, ok := lookupPref(key)
	if !ok {
		t.Fatalf("no pref %q", key)
	}
	return p.spec.Default
}

func TestHandleLinksLivesInItsSourceBlock(t *testing.T) {
	s, dir := open(t)
	on := true
	if _, err := s.Update(func(v *Settings) { v.ThunderstoreHandleLinks = &on }); err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Sources map[string]map[string]any `json:"sources"`
	}
	if err := json.Unmarshal(b, &doc); err != nil || doc.Sources["thunderstore"]["handleLinks"] != true {
		t.Fatalf("scopes: %v %s", err, b)
	}
	s2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Get().ThunderstoreHandleLinks; got == nil || !*got {
		t.Fatalf("reload = %v", got)
	}
}
