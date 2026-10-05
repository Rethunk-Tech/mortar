package savessvc

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestSaveGapSplitsMissingDisabledAndOlder(t *testing.T) {
	e := newSaveEnv(t)
	p, err := e.profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	e.alphaItem(t)
	if _, err := e.profiles.AddEntry("stardew", p.ID, "local-a", profile.Source{Kind: profile.KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	last := NewStore(t.TempDir())
	err = last.RecordRun("stardew", "Farm_1", "other", time.Now(), []PlayedMod{
		{ID: "smapi:A.Mod", Name: "Alpha", Version: "2.0.0"},
		{ID: "smapi:B.Mod", Name: "Beta", Version: "1.5.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{last: last, profiles: e.profiles}
	gap, err := svc.SaveGap("stardew", p.ID, "Farm_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(gap.Missing) != 1 || gap.Missing[0].ID != "smapi:B.Mod" || gap.Missing[0].Version != "1.5.0" {
		t.Fatalf("missing %#v", gap.Missing)
	}
	if len(gap.VersionOlder) != 1 || gap.VersionOlder[0].Have != "1.0.0" || len(gap.Disabled) != 0 {
		t.Fatalf("gap %#v", gap)
	}
	none, err := svc.SaveGap("stardew", p.ID, "Farm_9")
	if err != nil || len(none.Missing)+len(none.Disabled)+len(none.VersionOlder) != 0 {
		t.Fatalf("unplayed %#v %v", none, err)
	}
}

func TestSaveGapReportsSwitchedOffModAsDisabled(t *testing.T) {
	e := newSaveEnv(t)
	p, err := e.profiles.Create("stardew", "Main")
	if err != nil {
		t.Fatal(err)
	}
	e.alphaItem(t)
	if _, err := e.profiles.AddEntry("stardew", p.ID, "local-a", profile.Source{Kind: profile.KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.profiles.SetModEnabled("stardew", p.ID, "local-a", "smapi:A.Mod", false); err != nil {
		t.Fatal(err)
	}
	last := NewStore(t.TempDir())
	if err := last.RecordRun("stardew", "Farm_1", "other", time.Now(), []PlayedMod{{ID: "smapi:A.Mod", Name: "Alpha", Version: "1.0.0"}}); err != nil {
		t.Fatal(err)
	}
	gap, err := (&Service{last: last, profiles: e.profiles}).SaveGap("stardew", p.ID, "Farm_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(gap.Disabled) != 1 || gap.Disabled[0].ID != "smapi:A.Mod" || gap.Disabled[0].Version != "1.0.0" || len(gap.Missing) != 0 || len(gap.VersionOlder) != 0 {
		t.Fatalf("gap %#v", gap)
	}
}
