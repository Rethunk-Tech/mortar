package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestExtraFolderListsAndInstallsOnlyItsOwnFolders(t *testing.T) {
	e := newEnv(t)
	st, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), st)
	p := mustCreate(t, e, "P")
	if got, err := svc.ExtraFolderMods("stardew"); err != nil || len(got.Mods) != 0 {
		t.Fatalf("unset folder = %+v, %v", got, err)
	}
	extra := t.TempDir()
	writeFile(t, extra, "A/manifest.json", manifestJSON("me.a"))
	writeFile(t, extra, "Pack/B/manifest.json", manifestJSON("me.b"))
	setGamePref(t, st, "extraModsFolder", extra)
	got, err := svc.ExtraFolderMods("stardew")
	if err != nil || len(got.Mods) != 2 {
		t.Fatalf("listed = %+v, %v", got, err)
	}
	folder := ""
	for _, m := range got.Mods {
		if m.UniqueID == "me.b" {
			folder = m.Folder
		}
	}
	if folder != filepath.Join(extra, "Pack") {
		t.Fatalf("folder = %q", folder)
	}
	res, err := svc.InstallExtraFolderMod("stardew", p.ID, folder)
	if err != nil || len(res.Added) != 1 {
		t.Fatalf("install = %+v, %v", res, err)
	}
	for _, bad := range []string{t.TempDir(), filepath.Join(extra, "Pack", "B"), extra} {
		if _, err := svc.InstallExtraFolderMod("stardew", p.ID, bad); err == nil {
			t.Errorf("installed %s from outside the extra folder", bad)
		}
	}
}

func setGamePref(t *testing.T, st *settings.Store, key, value string) {
	t.Helper()
	var applyErr error
	if _, err := st.Update(func(s *settings.Settings) { applyErr = settings.ApplyKeyGame(s, key, value, settings.GameStardew) }); err != nil {
		t.Fatal(err)
	}
	if applyErr != nil {
		t.Fatal(applyErr)
	}
}
