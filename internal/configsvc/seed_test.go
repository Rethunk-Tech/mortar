package configsvc_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/configsvc"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestAPackagesShippedConfigIsEditableBeforeTheFirstLaunch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ps := profile.OpenIn(dir, store.OpenAt(filepath.Join(dir, "store")))
	const lc = "lethal-company"
	p, err := ps.Create(lc, "Friends")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "a.zip"), map[string]string{
		"manifest.json":          `{"name":"A","version_number":"1.0.0"}`,
		"config/ns.a.plugin.cfg": "[General]\nValue = shipped\n",
	})
	if _, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-A", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	s := &configsvc.Service{Profiles: ps}
	if err := s.Set(lc, p.ID, "", "ns.a.plugin.cfg", "General", "Value", "edited"); err != nil {
		t.Fatal(err)
	}
	if err := ps.SyncPackages(lc, p.ID); err != nil {
		t.Fatal(err)
	}
	root, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := fsx.ReadFile(filepath.Join(root, "BepInEx", "config", "ns.a.plugin.cfg"))
	if err != nil || !strings.Contains(string(b), "Value = edited") {
		t.Fatalf("after the launch's sync the file reads %q, %v; want the edit kept", b, err)
	}
}
