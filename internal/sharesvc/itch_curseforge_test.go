package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestAnItchModIsUnavailableUntilTheReceiverHoldsAFileFromThePage(t *testing.T) {
	ref := share.Ref{Itch: "someone/cool-mod"}
	m := (&resolver{}).itch(ref)
	if m.Site != SiteItch || m.State != StateUnavailable || m.Reason != ReasonItch || m.PageURL != "https://someone.itch.io/cool-mod" {
		t.Fatalf("mod = %+v", m)
	}
	held := &resolver{target: []profile.Entry{{Source: profile.Source{Kind: profile.KindItch, Name: "someone/cool-mod"}}}}
	if got := held.itch(ref); got.State != StateInstalled {
		t.Fatalf("a profile that holds the page's file has it installed: %+v", got)
	}
}

func TestACurseForgeModIsFetchedByHandWithoutAKey(t *testing.T) {
	ref := share.Ref{CurseForge: 309243, FileID: 555, Disabled: nil}
	keyless := &resolver{curseforgeUnavailable: func() string { return "Needs a CurseForge key" }}
	m := keyless.curseforge(ref)
	if m.Site != SiteCurseForge || m.State != StateUnavailable || m.Reason != ReasonCurseForgeKey || m.PageURL != "https://www.curseforge.com/projects/309243" {
		t.Fatalf("mod = %+v", m)
	}
	keyed := &resolver{curseforgeUnavailable: func() string { return "" }}
	if got := keyed.curseforge(ref); got.State != StateDownload || got.Package != "309243" || got.FileID != 555 {
		t.Fatalf("with a key the file downloads: %+v", got)
	}
	held := &resolver{target: []profile.Entry{{Source: profile.Source{Kind: profile.KindCurseForge, Name: "309243", FileID: 555}}}, curseforgeUnavailable: keyless.curseforgeUnavailable}
	if got := held.curseforge(ref); got.State != StateInstalled {
		t.Fatalf("a profile that holds the file has it installed: %+v", got)
	}
	stored := &resolver{
		game: "stardew", storedKeys: map[string]bool{}, curseforgeUnavailable: keyless.curseforgeUnavailable,
		stored: func(_, key string) bool { return key == store.PackageKey("curseforge:309243", "555") },
	}
	if got := stored.curseforge(ref); got.State != StateInstalled {
		t.Fatalf("a file already in the store is placed, not fetched: %+v", got)
	}
}

func TestACurseForgeModInstallsThroughTheCurseForgeRequestPath(t *testing.T) {
	m := (&resolver{curseforgeUnavailable: func() string { return "" }}).curseforge(share.Ref{CurseForge: 7, FileID: 9})
	r := requestFor("stardew", "p", m)
	if r.Kind != queue.KindInstall || r.Source != profile.KindCurseForge || r.Package != "7" || r.PackageFile != 9 {
		t.Fatalf("request = %+v", r)
	}
	e, ok := m.storedEntry()
	if !ok || e.source.Kind != profile.KindCurseForge || e.source.Name != "7" || e.source.FileID != 9 || e.key != store.PackageKey("curseforge:7", "9") {
		t.Fatalf("stored = %+v, %v", e, ok)
	}
}

func TestReplaceKnowsCurseForgeFiles(t *testing.T) {
	m := Mod{Site: SiteCurseForge, Package: "7", FileID: 9}
	e := profile.Entry{Key: "k", Source: profile.Source{Kind: profile.KindCurseForge, Name: "7", FileID: 9}}
	if modToken(m) == "" || modToken(m) != entryToken(e) {
		t.Fatalf("tokens %q %q", modToken(m), entryToken(e))
	}
}
