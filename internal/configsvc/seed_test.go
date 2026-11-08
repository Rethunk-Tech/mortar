package configsvc_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/configsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAPackagesShippedConfigIsEditableBeforeTheFirstLaunch(t *testing.T) {
	t.Parallel()
	const lc = "lethal-company"
	ps, p := testenv.LethalFriends(t)
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "a.zip"), map[string]string{
		"manifest.json":          `{"name":"A","version_number":"1.0.0"}`,
		"config/ns.a.plugin.cfg": "[General]\nValue = shipped\n",
	})
	if _, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-A", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	s := &configsvc.Service{Profiles: ps}
	if err := s.Set(t.Context(), lc, p.ID, "", "ns.a.plugin.cfg", "General", "Value", "edited"); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(lc, p.ID); err != nil {
		t.Fatal(err)
	}
	root, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := safefs.ReadFile(filepath.Join(root, "BepInEx", "config", "ns.a.plugin.cfg"))
	if err != nil || !strings.Contains(string(b), "Value = edited") {
		t.Fatalf("after the launch's sync the file reads %q, %v; want the edit kept", b, err)
	}
}

func TestASwitchedOffModsConfigStaysListedAndEditable(t *testing.T) {
	t.Parallel()
	const lc = "lethal-company"
	ps, p := testenv.LethalFriends(t)
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "a.zip"), map[string]string{
		"manifest.json": `{"name":"A","version_number":"1.0.0"}`,
		"config/A.cfg":  "[General]\nValue = shipped\n",
	})
	res, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-A", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	e := res.Profile.Entries[0]
	s := &configsvc.Service{Profiles: ps}
	if err := s.Set(t.Context(), lc, p.ID, "", "A.cfg", "General", "Value", "mine"); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.SetModEnabled(lc, p.ID, e.Key, e.Mods[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(lc, p.ID); err != nil {
		t.Fatal(err)
	}
	root, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := safefs.Stat(filepath.Join(root, profile.ConfigAsideDir, "A.cfg")); err != nil {
		t.Fatalf("switching the mod off set nothing aside: %v", err)
	}
	list, err := s.Mods(lc, p.ID)
	if err != nil || list.Without != 0 || len(list.Mods) != 1 || len(list.Mods[0].Files) != 1 {
		t.Fatalf("Mods = %+v, %v; want the switched-off mod listed with its set-aside file", list, err)
	}
	if err := s.Set(t.Context(), lc, p.ID, "", "A.cfg", "General", "Value", "edited off"); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.SetModEnabled(lc, p.ID, e.Key, e.Mods[0].ID, true); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(lc, p.ID); err != nil {
		t.Fatal(err)
	}
	b, err := safefs.ReadFile(filepath.Join(root, "BepInEx", "config", "A.cfg"))
	if err != nil || !strings.Contains(string(b), "Value = edited off") {
		t.Fatalf("switched back on, the file reads %q, %v; want the edit made while it was off", b, err)
	}
}
