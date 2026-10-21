package sharesvc

import (
	"context"
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/source/paradox"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestAParadoxModResolvesInstalledStoredOrDownloadAndIsNamedWhenTheSiteAnswers(t *testing.T) {
	ref := share.Ref{Paradox: "74604", FileID: 40, Version: "1.8.0"}
	m := (&resolver{}).paradox(t.Context(), ref)
	if m.Site != SiteParadox || m.State != StateDownload || m.Package != "74604" || m.FileID != 40 || m.Name != "Paradox mod 74604" || m.PageURL == "" {
		t.Fatalf("mod = %+v", m)
	}
	held := &resolver{target: []profile.Entry{{Source: profile.Source{Kind: profile.KindParadox, Name: "74604", FileID: 40, Version: "1.8.0"}}}}
	if got := held.paradox(t.Context(), ref); got.State != StateInstalled {
		t.Fatalf("a profile that holds the version has it installed: %+v", got)
	}
	stored := &resolver{
		game: "stellaris", storedKeys: map[string]bool{},
		stored: func(_, key string) bool { return key == store.PackageKey("paradox:74604", "1.8.0") },
	}
	if got := stored.paradox(t.Context(), ref); got.State != StateInstalled {
		t.Fatalf("a version already in the store is placed, not fetched: %+v", got)
	}
	named := &resolver{paradoxVersion: func(_ context.Context, id, version string) (paradox.Resolved, error) {
		return paradox.Resolved{Name: "Mod " + id + "/" + version}, nil
	}}
	if got := named.paradox(t.Context(), ref); got.Name != "Mod 74604/40" {
		t.Fatalf("name = %q", got.Name)
	}
	failing := &resolver{paradoxVersion: func(context.Context, string, string) (paradox.Resolved, error) {
		return paradox.Resolved{}, errors.New("offline")
	}}
	if got := failing.paradox(t.Context(), ref); got.Name != "Paradox mod 74604" {
		t.Fatalf("a failed lookup keeps the placeholder name: %q", got.Name)
	}
}

func TestAParadoxModInstallsThroughItsSourceRequestAndStoresUnderItsVersion(t *testing.T) {
	m := (&resolver{}).paradox(t.Context(), share.Ref{Paradox: "7", FileID: 9, Version: "2.0"})
	r := requestFor("stellaris", "p", m)
	if r.Kind != queue.KindInstall || r.Source != profile.KindParadox || r.Package != "7" || r.PackageFile != 9 {
		t.Fatalf("request = %+v", r)
	}
	e, ok := m.storedEntry()
	if !ok || e.source.Kind != profile.KindParadox || e.source.Name != "7" || e.source.FileID != 9 || e.key != store.PackageKey("paradox:7", "2.0") {
		t.Fatalf("stored = %+v, %v", e, ok)
	}
	entry := profile.Entry{Key: "k", Source: profile.Source{Kind: profile.KindParadox, Name: "7", FileID: 9}}
	if modToken(m) == "" || modToken(m) != entryToken(entry) {
		t.Fatalf("tokens %q %q", modToken(m), entryToken(entry))
	}
}
