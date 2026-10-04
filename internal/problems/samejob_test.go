package problems

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

func TestFootprintsJoinAssemblyWritesAndHarmonyReplaces(t *testing.T) {
	dll, err := fsx.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	profile := t.TempDir()
	if err := os.MkdirAll(filepath.Join(profile, "startup"), 0o700); err != nil {
		t.Fatal(err)
	}
	report := `{"schema":1,"replaces":{"A.Tools":["StardewValley.Farmer::updateCommon"]}}`
	if err := os.WriteFile(filepath.Join(profile, "startup", "20261003T000000Z.json"), []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	var mods []Installed
	for _, id := range []string{"A.Tools", "B.Tools"} {
		folder := filepath.Join(profile, "mods", id)
		if err := os.MkdirAll(folder, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "Mod.dll"), dll, 0o600); err != nil {
			t.Fatal(err)
		}
		mods = append(mods, fromDisk(Installed{Key: id, Folder: folder, Enabled: true, UniqueID: id, Name: id, EntryDll: "Mod.dll"}))
	}
	mods = append(mods, fromDisk(Installed{Key: "escape", Folder: profile, Enabled: true, UniqueID: "C.Escape", EntryDll: "../Mod.dll"}))

	fp := footprints(mods, launchsvc.LatestReplaces(profile))

	if len(fp) != 2 || !fp["a.tools"]["harmony:StardewValley.Farmer::updateCommon"] || fp["b.tools"]["harmony:StardewValley.Farmer::updateCommon"] {
		t.Fatalf("footprints = %v", fp)
	}
	if !fp["a.tools"]["StardewValley.Farmer::CurrentToolIndex"] || !fp["b.tools"]["StardewValley.Farmer::CurrentToolIndex"] {
		t.Fatalf("footprints = %v", fp)
	}
}

func TestSameJobWeighsMembersByHowFewModsWriteThem(t *testing.T) {
	fp := map[string]map[string]bool{}
	var mods []Installed
	add := func(id, author string, deps []string, members ...string) {
		set := map[string]bool{"Common::all": true}
		for _, m := range members {
			set[m] = true
		}
		fp[id] = set
		m := Installed{Key: "k-" + id, Enabled: true, UniqueID: id, Name: id, Author: author}
		for _, d := range deps {
			m.Dependencies = append(m.Dependencies, manifest.Dependency{UniqueID: d})
		}
		mods = append(mods, m)
	}
	add("a", "Ann", nil, "T::tool")
	add("b", "Bob", nil, "T::tool")
	add("c", "Cat", nil, "T::tool", "X::1", "X::2", "X::3")
	add("p1", "Pat", nil, "P::p")
	add("p2", " pat ", nil, "P::p")
	add("q1", "Quinn", nil, "Q::q")
	add("q2", "Quill", []string{"Q1"}, "Q::q")
	add("r1", "Ray", nil, "R::r")
	add("r2", "Rex", nil, "R::r")
	add("m1", "Mia", nil, "M::menu")
	add("m2", "Max", nil, "M::menu")
	for i := range 20 {
		if i < 11 {
			add(fmt.Sprint("filler", i), "", []string{"r1"}, fmt.Sprint("F::", i), "M::menu")
		} else {
			add(fmt.Sprint("filler", i), "", []string{"r1"}, fmt.Sprint("F::", i))
		}
	}
	mods = append(mods, Installed{Key: "off", UniqueID: "off"})
	fp["off"] = map[string]bool{"T::tool": true}

	got := sameJob(fp, mods)

	if len(got) != 2 || got[0].Key != "k-a" || got[1].Key != "k-b" || got[0].Kind != "sameJob" {
		t.Fatalf("rows = %+v", got)
	}
	if !slices.Equal(got[0].By, []ModRef{{Key: "k-b", Name: "b"}}) || got[0].Detail != "T.tool" {
		t.Fatalf("row = %+v", got[0])
	}
}

func TestSameJobOnSmallProfilesNeedsOneFootprintInsideTheOther(t *testing.T) {
	fp := map[string]map[string]bool{
		"a": {"T::t": true},
		"b": {"T::t": true, "U::u": true},
		"c": {"V::v": true, "W::w": true},
		"d": {"V::v": true, "X::x": true},
	}
	var mods []Installed
	for _, id := range []string{"a", "b", "c", "d"} {
		mods = append(mods, Installed{Key: id, Enabled: true, UniqueID: id, Name: id})
	}

	got := sameJob(fp, mods)

	if len(got) != 2 || got[0].Key != "a" || got[1].Key != "b" || got[1].Detail != "T.t" {
		t.Fatalf("rows = %+v", got)
	}
}

func TestShortMembersCapsAtThreeMostDistinctive(t *testing.T) {
	weight := map[string]float64{"N.A::a": 3, "harmony:N.B+Inner::b": 2, "N.C::c": 1, "N.D::d": 0.5}
	got := shortMembers(map[string]bool{"N.A::a": true, "harmony:N.B+Inner::b": true, "N.C::c": true, "N.D::d": true}, func(m string) float64 { return weight[m] })
	if got != "A.a, Inner.b, C.c, …" {
		t.Fatalf("got %q", got)
	}
}

func TestSameJobListsASmallerModALargerOneCovers(t *testing.T) {
	fp := map[string]map[string]bool{}
	var mods []Installed
	add := func(id string, members ...string) {
		fp[id] = map[string]bool{}
		for _, m := range members {
			fp[id][m] = true
		}
		mods = append(mods, Installed{Key: "k-" + id, Enabled: true, UniqueID: id, Name: id, Author: id})
	}
	var large []string
	for i := range 20 {
		large = append(large, fmt.Sprint("L::", i))
	}
	small := []string{"S::1", "S::2", "S::3", "S::4", "S::5"}
	hubbed := []string{"R::1", "R::2", "R::3", "R::4", "R::5"}
	add("large", slices.Concat(large, small, hubbed, []string{"One::x", "Hub::h"})...)
	add("small", small...)
	add("one", "One::x")
	add("hubbed", slices.Concat(hubbed, []string{"Hub::h"})...)
	for i := range 12 {
		add(fmt.Sprint("filler", i), fmt.Sprint("F::", i), "Hub::h")
	}
	for i := range 6 {
		add(fmt.Sprint("other", i), fmt.Sprint("O::", i))
	}

	got := sameJob(fp, mods)

	if len(got) != 1 || got[0].Key != "k-small" || !got[0].Covered || !slices.Equal(got[0].By, []ModRef{{Key: "k-large", Name: "large"}}) {
		t.Fatalf("rows = %+v", got)
	}
	if got[0].Detail != "S.1, S.2, S.3, …" {
		t.Fatalf("detail = %q", got[0].Detail)
	}
}
