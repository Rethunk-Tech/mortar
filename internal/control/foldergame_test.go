package control

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func folderProfile() profile.Profile {
	file := func(item, name string) profile.Entry {
		k := item + "#" + name
		return profile.Entry{Key: k, Item: item, File: name, Package: true, Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, k), Name: name}}}
	}
	tray := profile.Entry{Key: "pkg-1#/tray", Item: "pkg-1", Package: true, Tray: true, TrayFiles: []profile.TrayFile{{Rel: "H.trayitem"}},
		Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, "pkg-1#tray"), Name: "Tray files"}}}
	return profile.Profile{Name: "S", Entries: []profile.Entry{file("pkg-1", "a.package"), file("pkg-1", "b.package"), tray, file("pkg-2", "c.package")}}
}

func TestModRowsNameTheArchiveAndMarkTrayEntries(t *testing.T) {
	rows := modRows(folderProfile())
	if rows[0].Item != "pkg-1" || rows[0].File != "a.package" || rows[0].Tray || !rows[2].Tray {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestModIDsWithAHashAndArchiveNamesReachTheirEntries(t *testing.T) {
	p := folderProfile()
	refs, err := refsFor(p, []string{"folder:pkg-1#b.package"})
	if err != nil || len(refs) != 1 || refs[0].Key != "pkg-1#b.package" {
		t.Fatalf("typed id: %+v, %v", refs, err)
	}
	refs, err = refsFor(p, []string{"pkg-1#B.package"})
	if err != nil || len(refs) != 1 || refs[0].Key != "pkg-1#b.package" {
		t.Fatalf("untyped id: %+v, %v", refs, err)
	}
	keys, err := keysFor(p, []string{"pkg-1"})
	if err != nil || !slices.Equal(keys, []string{"pkg-1#a.package", "pkg-1#b.package", "pkg-1#/tray"}) {
		t.Fatalf("archive keys: %v, %v", keys, err)
	}
	if _, err := refsFor(p, []string{"pkg-9"}); err == nil {
		t.Fatal("an unknown name must be refused")
	}
}
