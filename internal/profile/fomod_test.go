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
	"github.com/Rethunk-Tech/mortar/internal/store"
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

// legacyFomod is a FOMOD archive whose folder has an empty destination, which the spec installs in the mod's folder and
// Mortar used to install under its own name.
func legacyFomod(id, dll string) map[string]string {
	return map[string]string{
		"fomod/ModuleConfig.xml": `<config><moduleName>M</moduleName><requiredInstallFiles><folder source="Main" destination=""/></requiredInstallFiles></config>`,
		"Main/manifest.json":     manifestJSON(id),
		"Main/m.dll":             dll,
	}
}

// installLegacy installs key as Mortar placed it before following the spec, with a config file the player wrote.
func installLegacy(t *testing.T, e env, id, key string) {
	t.Helper()
	if _, err := e.InstallFomod("stardew", id, key, Source{Kind: KindLocal, Name: key}, nil); err != nil {
		t.Fatal(err)
	}
	modsDir := e.mods(id)
	if _, err := e.updateMods("stardew", id, func(p *Profile, dir string) error {
		old := &p.Entries[len(p.Entries)-1]
		old.FomodSpec, old.Mods[0].Folder = false, "Main"
		if err := removeEntryFolders(modsDir, old.Key); err != nil {
			return err
		}
		return e.place("stardew", modsDir, *old)
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modsDir, key, "Main", "config.json"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// filesUnder lists the regular files below dir as sorted slash paths.
func filesUnder(dir string) string {
	var names []string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() {
			rel, _ := filepath.Rel(dir, p)
			names = append(names, filepath.ToSlash(rel))
		}
		return err
	})
	return strings.Join(names, ",")
}

// An entry placed by the destination rule Mortar used before the FOMOD spec's keeps that layout until it is installed
// again; the update then lays the new version out by the spec and carries the player's file to the mod's new folder,
// leaving nothing at the old place.
func TestAnEntryPlacedByTheOldFomodRuleUpdatesToTheSpecLayoutCleanly(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	e.item(t, "v1", legacyFomod("A.Main", "one"))
	e.item(t, "v2", legacyFomod("A.Main", "two"))
	installLegacy(t, e, p.ID, "v1")
	got, err := e.UpdateEntry("stardew", p.ID, "v1", "v2")
	if err != nil {
		t.Fatal(err)
	}
	ne := got.Entries[0]
	dir := liveEntryDir(e.mods(p.ID), ne.Key)
	dll, _ := os.ReadFile(filepath.Join(dir, "m.dll"))
	cfg, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if files := filesUnder(dir); !ne.FomodSpec || ne.Mods[0].Folder != "." || string(dll) != "two" || string(cfg) != "mine" ||
		files != "config.json,m.dll,manifest.json" {
		t.Fatalf("entry %+v, m.dll %q, config %q, files %v", ne, dll, cfg, files)
	}
}

func TestRelayoutMovesOldRuleEntriesWithTheirArchiveToTheSpecLayout(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "A")
	e.item(t, "kept", legacyFomod("A.Kept", "k"))
	e.item(t, "gone", legacyFomod("A.Gone", "g"))
	installLegacy(t, e, p.ID, "kept")
	installLegacy(t, e, p.ID, "gone")
	if err := e.items.Remove([]store.Ref{{Game: "stardew", Key: "gone"}}); err != nil {
		t.Fatal(err)
	}
	if got := e.RelayoutFomod(func(string) bool { return true }); got != (FomodRelayout{}) {
		t.Fatalf("a busy game was relaid out: %+v", got)
	}
	if got := e.RelayoutFomod(nil); got != (FomodRelayout{Moved: 1, NoArchive: 1}) {
		t.Fatalf("counts %+v", got)
	}
	after, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	kept, gone := after.Entries[0], after.Entries[1]
	keptFiles, goneFiles := filesUnder(liveEntryDir(e.mods(p.ID), "kept")), filesUnder(liveEntryDir(e.mods(p.ID), "gone"))
	if !kept.FomodSpec || kept.Mods[0].Folder != "." || kept.PreviousKey != "" || keptFiles != "config.json,m.dll,manifest.json" {
		t.Fatalf("kept %+v, files %s", kept, keptFiles)
	}
	if gone.FomodSpec || gone.Mods[0].Folder != "Main" || goneFiles != "Main/config.json,Main/m.dll,Main/manifest.json" {
		t.Fatalf("gone %+v, files %s", gone, goneFiles)
	}
}
