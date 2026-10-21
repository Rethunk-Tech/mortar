package packsvc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestExportModpackReimportsAsTheSamePackages(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "BepInEx", "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "BepInEx", "config", "ainavt.lc.lethalconfig.cfg"), []byte("[General]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ts := func(name, version string) profile.Entry {
		return profile.Entry{
			Source: profile.Source{Kind: profile.KindThunderstore, Name: name, Version: version},
			Mods:   []profile.Component{{ID: mod.NewID(mod.FormatThunderstore, name)}},
		}
	}
	off := ts("Alice-Quiet", "0.1.0")
	off.Disabled = []mod.ID{off.Mods[0].ID}
	ps := &fakeProfiles{dir: dir, list: []profile.Profile{{ID: "p1", Name: "Friday Night!", Entries: []profile.Entry{
		ts("AinaVT-LethalConfig", "1.4.6"),
		ts("notnotnotswipez-MoreCompany", "1.14.0"),
		off,
		{Source: profile.Source{Kind: profile.KindNexus, ModID: 7}, Mods: []profile.Component{{Name: "Nexus Thing"}}},
		{Source: profile.Source{Kind: profile.SourceMortar, Name: "Mortar"}},
	}}}}
	s := &Service{Profiles: ps}
	dest := filepath.Join(t.TempDir(), "pack.zip")
	res, err := s.ExportModpack("lethal-company", "p1", dest, ModpackOptions{Name: "Friday_Night", Version: "1.0.0", Configs: []string{"ainavt.lc.lethalconfig.cfg"}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.LeftOut, []string{"Nexus Thing"}) || !slices.Equal(res.Disabled, []string{"Alice-Quiet"}) || res.Configs != 1 {
		t.Fatalf("result %+v", res)
	}

	pv, err := s.read(context.Background(), dest)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range pv.Packages {
		got = append(got, p.Native+"-"+p.Version)
	}
	if pv.Name != "Friday_Night" || !slices.Equal(got, res.Dependencies) || len(got) != 2 || len(pv.Configs) != 1 {
		t.Fatalf("re-imported %+v, want %v", pv, res.Dependencies)
	}
}

func TestExportR2zReimportsAsTheSameModList(t *testing.T) {
	off := profile.Entry{
		Source: profile.Source{Kind: profile.KindThunderstore, Name: "Alice-Quiet", Version: "0.1.0"},
		Mods:   []profile.Component{{ID: mod.NewID(mod.FormatThunderstore, "Alice-Quiet")}},
	}
	off.Disabled = []mod.ID{off.Mods[0].ID}
	ps := &fakeProfiles{list: []profile.Profile{{ID: "p1", Name: "Friends", Entries: []profile.Entry{
		{Source: profile.Source{Kind: profile.KindThunderstore, Name: "AinaVT-LethalConfig", Version: "1.4.6"}},
		off,
	}}}}
	s := &Service{Profiles: ps}
	dest := filepath.Join(t.TempDir(), "friends.r2z")
	if _, err := s.ExportR2z("lethal-company", "p1", dest); err != nil {
		t.Fatal(err)
	}
	pv, err := s.read(context.Background(), dest)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(pv.Packages))
	for _, p := range pv.Packages {
		mark := ""
		if p.Disabled {
			mark = " off"
		}
		got = append(got, p.Native+"-"+p.Version+mark)
	}
	want := []string{"AinaVT-LethalConfig-1.4.6", "Alice-Quiet-0.1.0 off"}
	if pv.Name != "Friends" || !slices.Equal(got, want) {
		t.Fatalf("re-imported %+v", pv)
	}
}

func TestModpackFormSuggestsABumpAndChangelogAgainstTheLastExport(t *testing.T) {
	ts := func(name, version string) profile.Entry {
		return profile.Entry{
			Source: profile.Source{Kind: profile.KindThunderstore, Name: name, Version: version},
			Mods:   []profile.Component{{ID: mod.NewID(mod.FormatThunderstore, name)}},
		}
	}
	off := ts("Alice-Quiet", "0.1.0")
	off.Disabled = []mod.ID{off.Mods[0].ID}
	ps := &fakeProfiles{dir: t.TempDir(), list: []profile.Profile{{ID: "p1", Name: "Friends", Entries: []profile.Entry{ts("A-One", "1.0.0"), ts("B-Two", "1.0.0"), off}}}}
	s := &Service{Profiles: ps}
	first, err := s.ModpackForm("lethal-company", "p1")
	if err != nil || first.Options.Version != "1.0.0" || first.Options.Changelog != "" || !slices.Equal(first.Disabled, []string{"Alice-Quiet"}) {
		t.Fatalf("first form %+v, %v", first, err)
	}
	if _, err := s.ExportModpack("lethal-company", "p1", filepath.Join(t.TempDir(), "a.zip"), ModpackOptions{Name: "Friends", Version: "1.0.0", Author: "Me"}); err != nil {
		t.Fatal(err)
	}
	ps.list[0].Entries = []profile.Entry{ts("A-One", "1.1.0"), ts("C-Three", "2.0.0"), off}
	next, err := s.ModpackForm("lethal-company", "p1")
	if err != nil || next.Options.Version != "1.0.1" || next.Options.Author != "Me" || next.LastVersion != "1.0.0" {
		t.Fatalf("next form %+v, %v", next, err)
	}
	for _, want := range []string{"## 1.0.1", "- Updated A-One from 1.0.0 to 1.1.0", "- Added C-Three 2.0.0", "- Removed B-Two"} {
		if !strings.Contains(next.Options.Changelog, want) {
			t.Errorf("changelog lacks %q:\n%s", want, next.Options.Changelog)
		}
	}
	withOff, err := s.ModpackChangelog("lethal-company", "p1", "1.0.1", true)
	if err != nil || !strings.Contains(withOff, "- Added Alice-Quiet 0.1.0") {
		t.Errorf("switched-off package missing from %q, %v", withOff, err)
	}
	bad := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(bad, []byte("not a png"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, o := range []ModpackOptions{{Name: "x y", Version: "1.0.0"}, {Name: "x", Version: "1.0"}, {Name: "x", Version: "1.0.0", IconPath: bad}, {Name: "x", Version: "1.0.0", Configs: []string{"../x"}}} {
		if _, err := s.ExportModpack("lethal-company", "p1", filepath.Join(t.TempDir(), "b.zip"), o); err == nil {
			t.Errorf("exported with %+v", o)
		}
	}
}
