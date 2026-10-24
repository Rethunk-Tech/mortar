package profile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestInstallArchiveAsksForFomodThenInstallsChoices(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	xml, err := fsx.ReadFile(filepath.Join("..", "fomod", "testdata", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	z := buildZip(t, "fomod.zip", map[string]string{
		"fomod/ModuleConfig.xml": string(xml),
		"alpha/manifest.json":    manifestJSON("A.Alpha"),
		"beta/manifest.json":     manifestJSON("A.Beta"),
	})
	res, err := e.InstallArchive(t.Context(), "stardew", p.ID, z)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod == nil || res.Fomod.ModuleName != "Choose One" {
		t.Fatalf("want fomod ask, got %+v", res)
	}
	if len(res.Profile.Entries) != 0 {
		t.Fatalf("asked but already installed: %+v", res.Profile.Entries)
	}
	res, err = e.InstallFomod("stardew", p.ID, res.Fomod.Key, res.Fomod.Source.WithDisabled([]mod.ID{"smapi:A.Alpha"}), map[string]map[string][]string{
		"Options": {"Pack": {"Alpha"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod != nil || len(res.Added) != 1 || res.Added[0] != "A.Alpha" {
		t.Fatalf("install: %+v", res)
	}
	got, err := os.ReadFile(filepath.Join(e.mods(p.ID), "."+res.Profile.Entries[0].Key, "manifest.json"))
	if err != nil || string(got) != manifestJSON("A.Alpha") {
		t.Fatalf("layout %q %v", got, err)
	}
	if res.Profile.Entries[0].Fomod["Options"]["Pack"][0] != "Alpha" {
		t.Fatalf("stored choices: %+v", res.Profile.Entries[0].Fomod)
	}
	if len(res.Profile.Entries[0].Disabled) != 1 || res.Profile.Entries[0].Disabled[0] != "smapi:A.Alpha" {
		t.Fatalf("disabled mods: %+v", res.Profile.Entries[0].Disabled)
	}
}

func TestFomodReplayMismatchAsksAgain(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	xml, err := fsx.ReadFile(filepath.Join("..", "fomod", "testdata", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	e.item(t, "local-fomod", map[string]string{
		"fomod/ModuleConfig.xml": string(xml),
		"alpha/manifest.json":    manifestJSON("A.Alpha"),
		"beta/manifest.json":     manifestJSON("A.Beta"),
	})
	res, err := e.InstallFomod("stardew", p.ID, "local-fomod", Source{Kind: KindLocal, Name: "x"}, map[string]map[string][]string{
		"Options": {"Pack": {"Gone"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fomod == nil {
		t.Fatal("vanished choice should ask again")
	}
}

func TestFomodUpdateReplaysOnlyUnchangedOptions(t *testing.T) {
	t.Parallel()
	raw, err := fsx.ReadFile(filepath.Join("..", "fomod", "testdata", "choose-one.xml"))
	if err != nil {
		t.Fatal(err)
	}
	base := string(raw)
	extra := `<group name="Extras" type="SelectAny"><plugins><plugin name="Gamma"><description/><files/>` +
		`<typeDescriptor><type name="Optional"/></typeDescriptor></plugin></plugins></group></optionalFileGroups>`
	alpha := map[string]map[string][]string{"Options": {"Pack": {"Alpha"}}}
	for _, tc := range []struct {
		name, xml string
		want      map[string]map[string][]string
	}{
		{"unchanged", base, nil},
		{"new group", strings.Replace(base, "</optionalFileGroups>", extra, 1), alpha},
		{"plugin gone", strings.Replace(base, `name="Alpha"`, `name="Alpha2"`, 1), map[string]map[string][]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			p, err := e.Create("stardew", "A")
			if err != nil {
				t.Fatal(err)
			}
			files := func(xml string) map[string]string {
				return map[string]string{
					"fomod/ModuleConfig.xml": xml,
					"alpha/manifest.json":    manifestJSON("A.Alpha"),
					"beta/manifest.json":     manifestJSON("A.Beta"),
				}
			}
			e.item(t, "v1", files(base))
			e.item(t, "v2", files(tc.xml))
			if _, err := e.InstallFomod("stardew", p.ID, "v1", Source{Kind: KindLocal, Name: "x"}, alpha); err != nil {
				t.Fatal(err)
			}
			src, ask, need, err := e.installAsk("stardew", p.ID, "v2", Source{Kind: KindLocal, Name: "x"})
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == nil {
				if need || !reflect.DeepEqual(src.fomodMap(), alpha) {
					t.Fatalf("install should replay silently: need=%v choices=%v", need, src.fomodMap())
				}
			} else if !need || !ask.Changed || ask.OldKey != "v1" || !reflect.DeepEqual(ask.Choices, tc.want) {
				t.Fatalf("install should re-offer: need=%v ask=%+v", need, ask)
			}
			_, err = e.UpdateEntry("stardew", p.ID, "v1", "v2")
			need2, asked := errors.AsType[*NeedChoicesError](err)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("update should replay silently: %v", err)
				}
			} else if !asked || !need2.Ask.Changed || !reflect.DeepEqual(need2.Ask.Choices, tc.want) {
				t.Fatalf("update should re-offer: %v", err)
			}
		})
	}
}

// An entry placed by the destination rule Mortar used before the FOMOD spec's keeps that layout until it is installed
// again; the update then lays the new version out by the spec and carries the player's file to the mod's new folder,
// leaving nothing at the old place.
func TestAnEntryPlacedByTheOldFomodRuleUpdatesToTheSpecLayoutCleanly(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	files := func(dll string) map[string]string {
		return map[string]string{
			"fomod/ModuleConfig.xml": `<config><moduleName>M</moduleName><requiredInstallFiles><folder source="Main" destination=""/></requiredInstallFiles></config>`,
			"Main/manifest.json":     manifestJSON("A.Main"),
			"Main/m.dll":             dll,
		}
	}
	e.item(t, "v1", files("one"))
	e.item(t, "v2", files("two"))
	if _, err := e.InstallFomod("stardew", p.ID, "v1", Source{Kind: KindLocal, Name: "x"}, nil); err != nil {
		t.Fatal(err)
	}
	modsDir := e.mods(p.ID)
	if _, err := e.updateMods("stardew", p.ID, func(p *Profile, dir string) error {
		old := &p.Entries[0]
		old.FomodSpec, old.Mods[0].Folder = false, "Main"
		if err := removeEntryFolders(modsDir, old.Key); err != nil {
			return err
		}
		return e.place("stardew", modsDir, *old)
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modsDir, "v1", "Main", "config.json"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := e.UpdateEntry("stardew", p.ID, "v1", "v2")
	if err != nil {
		t.Fatal(err)
	}
	ne := got.Entries[0]
	dir := liveEntryDir(modsDir, ne.Key)
	var names []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			rel, _ := filepath.Rel(dir, p)
			names = append(names, filepath.ToSlash(rel))
		}
		return err
	})
	dll, _ := os.ReadFile(filepath.Join(dir, "m.dll"))
	cfg, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if !ne.FomodSpec || ne.Mods[0].Folder != "." || string(dll) != "two" || string(cfg) != "mine" ||
		strings.Join(names, ",") != "config.json,m.dll,manifest.json" {
		t.Fatalf("entry %+v, m.dll %q, config %q, files %v", ne, dll, cfg, names)
	}
}
