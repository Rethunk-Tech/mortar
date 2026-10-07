package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
)

type fakeCF struct {
	res curseforge.Resolved
	err error
}

func (fakeCF) Unavailable() string { return "" }

func (f fakeCF) Resolve(context.Context, string, string) (curseforge.Resolved, error) {
	return f.res, f.err
}

func cfUpdate() (framework.Mod, []Update) {
	x := inst("nexus-20619-123175", "Example.Bush", "1.0.0", true)
	x.SourceKind, x.UpdateKeys = profile.KindNexus, []string{"Nexus:20619", "CurseForge:998265"}
	return x, []Update{{Key: x.Key, ID: x.ModID(), Installed: "1.0.0", Version: "1.0.1", Source: "CurseForge", URL: "https://www.curseforge.com/stardewvalley/mods/bush"}}
}

func TestCurseForgeKeyUpdateQueuesTheNewestFileAndSwitches(t *testing.T) {
	x, ups := cfUpdate()
	s := &Service{}
	got := s.switchUpdates(t.Context(), fakeCF{res: curseforge.Resolved{Version: "Bush 1.0.1", VersionID: "777"}}, []framework.Mod{x}, ups)
	if len(got) != 1 || got[0].Package != "998265" || got[0].PackageSource != "curseforge" || got[0].PackageVersion != "777" ||
		got[0].Version != "1.0.1" || !got[0].Switch || got[0].FromSource == "" {
		t.Fatalf("updates = %+v", got)
	}
}

func TestCurseForgeKeyUpdateWithoutANewerFileIsDropped(t *testing.T) {
	x, ups := cfUpdate()
	got := (&Service{}).switchUpdates(t.Context(), fakeCF{res: curseforge.Resolved{Version: "Bush 1.0.0", VersionID: "700"}}, []framework.Mod{x}, ups)
	if len(got) != 0 {
		t.Fatalf("updates = %+v, want none", got)
	}
}

func TestCurseForgeKeyUpdateOfAForbiddenProjectIsFlagged(t *testing.T) {
	x, ups := cfUpdate()
	got := (&Service{}).switchUpdates(t.Context(), fakeCF{err: &curseforge.NotDistributableError{Mod: "Bush", PageURL: "https://x"}}, []framework.Mod{x}, ups)
	if len(got) != 1 || !got[0].NotDistributable || got[0].Package != "" {
		t.Fatalf("updates = %+v", got)
	}
}
