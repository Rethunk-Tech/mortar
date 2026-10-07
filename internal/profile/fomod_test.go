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
