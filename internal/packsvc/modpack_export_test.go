package packsvc

import (
	"context"
	"os"
	"path/filepath"
	"slices"
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
	res, err := s.ExportModpack("lethal-company", "p1", dest, true)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(res.LeftOut, []string{"Nexus Thing"}) || !slices.Equal(res.Disabled, []string{"Alice-Quiet"}) || res.Configs != 1 {
		t.Fatalf("result %+v", res)
	}

	pv, err := s.Preview(context.Background(), Source{Path: dest})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range pv.Packages {
		got = append(got, p.Native+"-"+p.Version)
	}
	if pv.Name != "Friday_Night" || !slices.Equal(got, res.Dependencies) || len(got) != 2 || pv.Configs != 1 {
		t.Fatalf("re-imported %+v, want %v", pv, res.Dependencies)
	}
}
