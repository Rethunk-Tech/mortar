package problems

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/launchsvc"
)

func TestPatchHintsFlagModsReplacingTheSameMethods(t *testing.T) {
	report := `{"schema":1,"replaces":{
		"a.hud":["StardewValley.Game1::drawHUD","StardewValley.Game1::draw","StardewValley.Farmer+Sub::x","Hub::h1","Hub::h2"],
		"B.HUD":["StardewValley.Game1::drawHUD","StardewValley.Game1::draw","StardewValley.Farmer+Sub::x","Other::y","Other::z"],
		"C.One":["StardewValley.Game1::draw","C::c"],
		"D.Hub":["Hub::h1","Hub::h2"],
		"E.Hub":["Hub::h1","Hub::h2"],
		"I.Hub":["Hub::h1","Hub::h2"],
		"F.Big":["Big::1","Big::2","Big::3","Big::4","Big::5"],
		"G.Big":["Big::1","Big::2","G::3","G::4","G::5"],
		"H.Off":["StardewValley.Game1::drawHUD","StardewValley.Game1::draw"]}}`
	profile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(profile, "startup"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "startup", "20261003T000000Z.json"), []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "startup", "20261003T000000Z.samples.json"), []byte(`{"schema":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	mods := []Installed{{Key: "h", UniqueID: "H.Off"}}
	for _, id := range []string{"A.Hud", "b.hud", "C.One", "D.Hub", "E.Hub", "I.Hub", "F.Big", "G.Big"} {
		mods = append(mods, Installed{Key: "k-" + id, UniqueID: id, Name: id, Enabled: true})
	}

	got := patchHints(launchsvc.LatestReplaces(profile), mods)

	if len(got) != 2 {
		t.Fatalf("hints = %+v", got)
	}
	a, b := got[0], got[1]
	if a.Kind != "patches" || a.Key != "k-A.Hud" || len(a.By) != 1 || a.By[0].Key != "k-b.hud" || b.Key != "k-b.hud" || b.By[0].Name != "A.Hud" {
		t.Fatalf("hints = %+v", got)
	}
	if want := "Game1.draw, Game1.drawHUD, Sub.x"; a.Detail != want {
		t.Fatalf("detail = %q, want %q", a.Detail, want)
	}
}

func TestPatchHintsWithoutReportAreEmpty(t *testing.T) {
	if got := patchHints(launchsvc.LatestReplaces(t.TempDir()), []Installed{{Key: "a", UniqueID: "A", Enabled: true}}); got != nil {
		t.Fatalf("hints = %+v", got)
	}
}

func TestShortMethodsCapsAtThree(t *testing.T) {
	got := shortMethods(map[string]bool{"N.A::a": true, "N.B::b": true, "N.C::c": true, "N.D::d": true})
	if got != "A.a, B.b, C.c, …" {
		t.Fatalf("got %q", got)
	}
}
