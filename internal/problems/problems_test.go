package problems

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

type fakeMeta struct {
	refs       map[string][]meta.Ref
	pages      map[int]meta.Page
	compat     map[string]meta.UpdateResult
	lookupErr  error
	pageErr    error
	updatesOff bool
}

func (f fakeMeta) Lookup(_ context.Context, id string) ([]meta.Ref, error) {
	return f.refs[id], f.lookupErr
}

func (f fakeMeta) Page(_ context.Context, id int) (meta.Page, error) {
	if f.pageErr != nil {
		return meta.Page{}, f.pageErr
	}
	return f.pages[id], nil
}

func (f fakeMeta) CheckUpdates(_ context.Context, req meta.UpdateRequest) []meta.UpdateResult {
	out := make([]meta.UpdateResult, len(req.Mods))
	for i, m := range req.Mods {
		out[i] = meta.UpdateResult{ID: m.ID}
		if r, ok := f.compat[m.ID]; ok && !f.updatesOff {
			out[i] = r
			out[i].ID = m.ID
		}
		out[i].Known = !f.updatesOff
	}
	return out
}

func mod(key, id, version string, enabled bool, deps ...manifest.Dependency) Installed {
	m := Installed{Key: key, SourceKind: profile.KindLocal, Enabled: enabled}
	m.Name, m.UniqueID, m.Version, m.Dependencies = id, id, version, deps
	return m
}

func req(id, minimum string) manifest.Dependency {
	return manifest.Dependency{UniqueID: id, MinimumVersion: minimum, Required: true}
}

func TestMissingKinds(t *testing.T) {
	cases := []struct {
		name string
		mods []Installed
		want []string // reasons
	}{
		{"absent", []Installed{mod("a", "A", "1.0", true, req("B", ""))}, []string{"absent"}},
		{"present", []Installed{mod("a", "A", "1.0", true, req("B", "")), mod("b", "B", "1.0", true)}, nil},
		{"disabled", []Installed{mod("a", "A", "1.0", true, req("B", "1.0")), mod("b", "B", "1.0", false)}, []string{"disabled"}},
		{"outdated", []Installed{mod("a", "A", "1.0", true, req("B", "2.0")), mod("b", "B", "1.5", true)}, []string{"outdated"}},
		{"case-insensitive id", []Installed{mod("a", "A", "1.0", true, req("b", "")), mod("b", "B", "1.0", true)}, nil},
		{"prerelease below release", []Installed{mod("a", "A", "1.0", true, req("B", "1.2")), mod("b", "B", "1.2-beta.1", true)}, []string{"outdated"}},
		{"release meets prerelease minimum", []Installed{mod("a", "A", "1.0", true, req("B", "1.2-beta.1")), mod("b", "B", "1.2", true)}, nil},
		{"prerelease meets lower prerelease", []Installed{mod("a", "A", "1.0", true, req("B", "1.2-alpha")), mod("b", "B", "1.2-beta", true)}, nil},
		{"unparseable version is not flagged", []Installed{mod("a", "A", "1.0", true, req("B", "1.2")), mod("b", "B", "weird", true)}, nil},
		{"optional ignored", []Installed{mod("a", "A", "1.0", true, manifest.Dependency{UniqueID: "B", Required: false})}, nil},
		{"disabled dependent ignored", []Installed{mod("a", "A", "1.0", false, req("B", ""))}, nil},
		{"one of two copies satisfies", []Installed{
			mod("a", "A", "1.0", true, req("B", "2.0")), mod("b1", "B", "1.0", true), mod("b2", "B", "2.0", true),
		}, []string{}},
	}
	for _, c := range cases {
		got := Check(context.Background(), fakeMeta{}, Environment{}, c.mods)
		var reasons []string
		for _, m := range got.Missing {
			reasons = append(reasons, m.Reason)
		}
		want := c.want
		if c.name == "one of two copies satisfies" {
			want = nil
			if len(got.Duplicates) != 1 {
				t.Errorf("%s: duplicates = %d", c.name, len(got.Duplicates))
			}
		}
		if !reflect.DeepEqual(reasons, want) {
			t.Errorf("%s: reasons = %v, want %v", c.name, reasons, want)
		}
	}
}

func TestContentPackForIsRequired(t *testing.T) {
	m := mod("a", "A", "1.0", true)
	m.Dependencies = []manifest.Dependency{{UniqueID: "Pathoschild.ContentPatcher", MinimumVersion: "2.0", Required: true}}
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{m})
	if len(got.Missing) != 1 || got.Missing[0].UniqueID != "Pathoschild.ContentPatcher" {
		t.Fatalf("missing = %+v", got.Missing)
	}
}

func file(id int64, typ, version string) meta.File {
	return meta.File{ID: id, Type: typ, FileName: "B-" + version + ".zip", Version: version, Mods: []meta.Mod{{UniqueID: "B", Version: version}}}
}

func TestWhere(t *testing.T) {
	pages := map[int]meta.Page{
		10: {ID: 10, Name: "Old host", PageURL: "https://n/10", Downloads: []meta.File{file(1, "MAIN", "1.0"), file(2, "OPTIONAL", "1.5")}},
		20: {ID: 20, Name: "New host", PageURL: "https://n/20", Downloads: []meta.File{file(3, "MAIN", "2.0"), file(4, "MAIN", "2.5"), file(5, "MAIN", "1.0")}},
	}
	dependent := func(keys ...string) []Installed {
		a := mod("a", "A", "1.0", true, req("B", "2.0"))
		a.UpdateKeys = keys
		return []Installed{a}
	}
	both := []meta.Ref{{Site: "Nexus", ID: 10}, {Site: "Nexus", ID: 20}}
	cases := []struct {
		name    string
		meta    fakeMeta
		mods    []Installed
		want    *Ref
		unknown bool
	}{
		{
			"highest version page, newest satisfying main file",
			fakeMeta{refs: map[string][]meta.Ref{"B": both}, pages: pages},
			dependent(),
			&Ref{Site: "Nexus", PageID: 20, PageName: "New host", URL: "https://n/20", FileID: 4, FileName: "B-2.5.zip", Version: "2.5"}, false,
		},
		{
			"dependent's update key wins",
			fakeMeta{refs: map[string][]meta.Ref{"B": both}, pages: pages},
			dependent("Nexus:10@x"),
			&Ref{Site: "Nexus", PageID: 10, PageName: "Old host", URL: "https://n/10"}, false,
		},
		{
			"curseforge only",
			fakeMeta{refs: map[string][]meta.Ref{"B": {{Site: "CurseForge", ID: 7}}}},
			dependent(),
			&Ref{Site: "CurseForge", PageID: 7, URL: "https://www.curseforge.com/projects/7"}, false,
		},
		{
			"github repo from SMAPI's API, ahead of other sites",
			fakeMeta{refs: map[string][]meta.Ref{"B": {{Site: "CurseForge", ID: 7}}}, compat: map[string]meta.UpdateResult{"B": {GitHubRepo: "me/b"}}},
			dependent(),
			&Ref{Site: "GitHub", GitHub: "me/b", URL: "https://github.com/me/b"}, false,
		},
		{"not in dataset", fakeMeta{}, dependent(), nil, false},
		{"dataset unreachable", fakeMeta{lookupErr: errors.New("offline")}, dependent(), nil, true},
		{
			"page fetch fails, link only",
			fakeMeta{refs: map[string][]meta.Ref{"B": {{Site: "Nexus", ID: 20}}}, pageErr: errors.New("offline")},
			dependent(),
			&Ref{Site: "Nexus", PageID: 20, URL: "https://www.nexusmods.com/stardewvalley/mods/20"}, true,
		},
	}
	for _, c := range cases {
		got := Check(context.Background(), c.meta, Environment{}, c.mods)
		if len(got.Missing) != 1 || !reflect.DeepEqual(got.Missing[0].Where, c.want) || got.Unknown != c.unknown {
			t.Errorf("%s: where = %+v, unknown = %v; want %+v, %v", c.name, got.Missing, got.Unknown, c.want, c.unknown)
		}
	}
}

func TestDuplicates(t *testing.T) {
	nexus := mod("n", "S", "2.0", true)
	nexus.SourceKind = "nexus"
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{
		mod("x", "L", "1.0", true, req("S", "2.0")),
		mod("o", "S", "1.0", true),
		nexus,
		mod("off", "S", "3.0", false),
	})
	if len(got.Duplicates) != 1 {
		t.Fatalf("duplicates = %+v", got.Duplicates)
	}
	c := got.Duplicates[0].Copies
	if len(c) != 2 || c[0].Key != "o" || c[0].Newest || c[0].Needed == nil || len(c[0].TooOld) != 1 || !c[1].Newest || !c[1].Nexus || len(c[1].Needed) != 1 {
		t.Fatalf("copies = %+v", c)
	}
	if len(got.Missing) != 0 {
		t.Fatalf("the newer copy satisfies the dependency, got %+v", got.Missing)
	}
}

func TestNexusFilesInDuplicate(t *testing.T) {
	a := mod("nexus-21788-116403", "SVE.WorldMap", "1.0", true)
	a.SourceKind = "nexus"
	b := mod("nexus-21788-175477", "SVE.WorldMap", "1.0", true)
	b.SourceKind = "nexus"
	got := Check(context.Background(), fakeMeta{pages: map[int]meta.Page{21788: {
		Downloads: []meta.File{
			{ID: 116403, Type: "MAIN", FileName: "WorldMap-1.0.zip", Version: "1.0"},
			{ID: 175477, Type: "MAIN", FileName: "WorldMap-2.0.zip", Version: "2.0"},
		},
	}}}, Environment{}, []Installed{a, b})
	if len(got.Duplicates) != 1 || len(got.Duplicates[0].NexusFiles) != 2 {
		t.Fatalf("nexus files = %+v", got.Duplicates)
	}
	if !got.Duplicates[0].NexusFiles[0].Remove || got.Duplicates[0].NexusFiles[1].Remove {
		t.Fatalf("remove flags = %+v", got.Duplicates[0].NexusFiles)
	}
}

func TestOptionalNexusFileDuplicateIsInformational(t *testing.T) {
	a := mod("nexus-21788-116403", "SVE.WorldMap", "1.0", true)
	a.SourceKind = "nexus"
	b := mod("nexus-21788-175477", "SVE.WorldMap", "1.0", true)
	b.SourceKind = "nexus"
	got := Check(context.Background(), fakeMeta{pages: map[int]meta.Page{21788: {
		Downloads: []meta.File{
			{ID: 116403, Type: "MAIN", FileName: "WorldMap-1.0.zip", Version: "1.0"},
			{ID: 175477, Type: "MISCELLANEOUS", FileName: "WorldMap-addon.zip", Version: "1.0"},
		},
	}}}, Environment{}, []Installed{a, b})
	if len(got.Duplicates) != 1 || !got.Duplicates[0].NexusOptional || got.Count() != 0 {
		t.Fatalf("optional duplicate = %+v, count = %d", got.Duplicates, got.Count())
	}
}

func TestBroken(t *testing.T) {
	m := fakeMeta{compat: map[string]meta.UpdateResult{
		"A": {Compatibility: "Broken", BrokeIn: "Stardew Valley 1.6"},
		"B": {Compatibility: "Ok"},
		"C": {Compatibility: "Obsolete"},
		"D": {Compatibility: "Broken"},
	}}
	mods := []Installed{mod("a", "A", "1", true), mod("b", "B", "1", true), mod("c", "C", "1", true), mod("d", "D", "1", false)}
	got := Check(context.Background(), m, Environment{}, mods)
	want := []Broken{
		{Key: "a", UniqueID: "A", Name: "A", Status: "broken", BrokeIn: "Stardew Valley 1.6"},
		{Key: "c", UniqueID: "C", Name: "C", Status: "obsolete"},
	}
	if !reflect.DeepEqual(got.Broken, want) || got.Unknown {
		t.Fatalf("broken = %+v, unknown = %v", got.Broken, got.Unknown)
	}
	off := Check(context.Background(), fakeMeta{updatesOff: true}, Environment{}, mods)
	if len(off.Broken) != 0 || !off.Unknown {
		t.Fatalf("offline: %+v", off)
	}
}

func TestCheckPopulatesTimings(t *testing.T) {
	got := Check(context.Background(), fakeMeta{}, Environment{}, []Installed{mod("a", "A", "1.0", true)})
	names := map[string]bool{}
	for _, tmg := range got.Timings {
		names[tmg.Name] = true
	}
	for _, name := range []string{"contentPatcher", "conflicts", "requirements", "others"} {
		if !names[name] {
			t.Fatalf("missing timing %q in %+v", name, got.Timings)
		}
	}
}

func TestCountShowsConflictsBetweenTheSameModsOnce(t *testing.T) {
	seasonal := func(target string) AssetConflict {
		return AssetConflict{Kind: "edit", Target: target, PackIDs: []string{"B", "A"}, WinnerName: "unclear"}
	}
	r := Result{AssetConflicts: []AssetConflict{
		seasonal("loosesprites/map"), seasonal("loosesprites/map_fall"),
		{Kind: "edit", Target: "maps/forest", PackIDs: []string{"A", "C"}, WinnerName: "unclear"},
		{Kind: "edit", Target: "x", PackIDs: []string{"A", "B"}, Cosmetic: true},
	}}
	if got := r.Count(); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}
}
