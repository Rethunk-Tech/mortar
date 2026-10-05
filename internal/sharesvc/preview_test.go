package sharesvc

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

type fakeMeta struct {
	refs    map[string][]meta.Ref
	pages   map[int]meta.Page
	broken  map[string]string
	coll    meta.Collection
	collErr error
}

func (f fakeMeta) Lookup(_ context.Context, id string) ([]meta.Ref, error) {
	return f.refs[strings.ToLower(id)], nil
}

func (f fakeMeta) Page(_ context.Context, id int) (meta.Page, error) {
	p, ok := f.pages[id]
	if !ok {
		return meta.Page{}, errors.New("no page")
	}
	return p, nil
}

func (f fakeMeta) CheckUpdates(_ context.Context, req meta.UpdateRequest) []meta.UpdateResult {
	out := make([]meta.UpdateResult, len(req.Mods))
	for i, m := range req.Mods {
		out[i] = meta.UpdateResult{ID: m.ID, Known: true, Compatibility: f.broken[m.ID], BrokeIn: "1.6.15"}
	}
	return out
}

func dsFile(id int64, version string, mods ...meta.Mod) meta.File {
	return meta.File{ID: id, Type: "Main", Version: version, SizeInBytes: 4096, Mods: mods}
}

func page(id int, name string, files ...meta.File) meta.Page {
	return meta.Page{ID: id, Name: name, Author: "someone", Downloads: files}
}

func nf(id int, version, category string, primary bool) nexus.File {
	return nexus.File{FileID: id, FileName: "f.zip", Version: version, Category: category, SizeKB: 10, IsPrimary: primary}
}

func testResolver(files map[int][]nexus.File, target []profile.Entry) *resolver {
	dep := meta.Mod{UniqueID: "B.Dep", Version: "1.2"}
	fm := fakeMeta{
		refs: map[string][]meta.Ref{"b.dep": {{Site: "Nexus", ID: 900}}},
		pages: map[int]meta.Page{
			100:  page(100, "Main Mod", dsFile(1, "1.0", meta.Mod{UniqueID: "A.Main", Version: "1.0", Dependencies: []meta.Dependency{{UniqueID: "B.Dep", MinimumVersion: "1.0", Required: true}}})),
			200:  page(200, "Gone File", dsFile(2, "2.0", meta.Mod{UniqueID: "A.Gone", Version: "2.0"}), dsFile(22, "2.0", meta.Mod{UniqueID: "A.Gone", Version: "2.0"})),
			700:  page(700, "Later Mod", dsFile(70, "1.0", meta.Mod{UniqueID: "A.Later", Version: "1.0"})),
			900:  page(900, "Dependency", dsFile(9, "1.2", dep)),
			1000: page(1000, "Broken Mod", dsFile(10, "1.0", meta.Mod{UniqueID: "A.Broken", Name: "Broken Mod", Version: "1.0"})),
		},
		broken: map[string]string{"A.Broken": "broken"},
	}
	return &resolver{
		meta: fm, signedIn: true, premium: false, target: target, env: stardewEnv,
		files: func(_ context.Context, _ nexus.Title, id int) ([]nexus.File, error) {
			if f, ok := files[id]; ok {
				return f, nil
			}
			return nil, &nexus.StatusError{Code: 404}
		},
	}
}

func byKey(mods []Mod, want Mod) Mod {
	for _, m := range mods {
		if m.ModID == want.ModID && m.Repo == want.Repo {
			return m
		}
	}
	return Mod{}
}

func TestResolveGroupsAndSubstitutes(t *testing.T) {
	files := map[int][]nexus.File{
		100:  {nf(1, "1.0", "MAIN", true)},
		200:  {nf(22, "2.0", "MAIN", false), nf(23, "3.0", "OPTIONAL", true)},
		300:  {nf(3, "1.5", "ARCHIVED", false), nf(33, "1.5", "MAIN", false), nf(34, "2.5", "MAIN", true)},
		400:  {nf(44, "9.0", "MAIN", true)},
		600:  {nf(6, "1.0", "MAIN", true)},
		700:  {nf(7, "1.0", "MAIN", true)},
		800:  {nf(88, "1.0", "OPTIONAL", false)},
		900:  {nf(9, "1.2", "MAIN", true)},
		1000: {nf(10, "1.0", "MAIN", true)},
	}
	target := []profile.Entry{{Key: "nexus-600-6", Source: profile.Source{Kind: profile.KindNexus, ModID: 600, FileID: 6}, Mods: []profile.Component{{ID: "smapi:A.Have", Name: "Have"}}}}
	r := testResolver(files, target)
	refs := []share.Ref{
		{ModID: 100, FileID: 1},
		{ModID: 200, FileID: 2},
		{ModID: 300, FileID: 3},
		{ModID: 400, FileID: 4},
		{ModID: 500, FileID: 5},
		{ModID: 600, FileID: 6},
		{ModID: 700, FileID: 7},
		{GitHub: "o/r@v1/a.zip"},
		{ModID: 800, FileID: 8},
		{ModID: 1000, FileID: 10},
	}
	mods, probs := r.resolve(t.Context(), refs)

	tests := []struct {
		name     string
		im       Mod
		state    string
		file     int
		differ   bool
		reason   string
		unverify bool
	}{
		{"download", Mod{ModID: 100}, StateDownload, 1, false, "", false},
		{"deleted file: MAIN of the same version", Mod{ModID: 200}, StateDownload, 22, true, "", false},
		{"archived file: MAIN of the same version, not primary", Mod{ModID: 300}, StateLater, 33, true, "", false},
		{"no same version: primary", Mod{ModID: 400}, StateLater, 44, true, "", false},
		{"mod removed", Mod{ModID: 500}, StateUnavailable, 5, false, ReasonRemoved, false},
		{"already in the profile", Mod{ModID: 600}, StateInstalled, 6, false, "", false},
		{"dataset lacks the file", Mod{ModID: 700}, StateLater, 7, false, "", false},
		{"nothing to substitute", Mod{ModID: 800}, StateUnavailable, 8, false, ReasonNoFile, false},
		{"github", Mod{Repo: "o/r"}, StateDownload, 0, false, "", true},
	}
	for _, tt := range tests {
		got := byKey(mods, tt.im)
		if got.State != tt.state || got.FileID != tt.file || got.Different != tt.differ || got.Reason != tt.reason || got.Unverified != tt.unverify {
			t.Errorf("%s: got %+v", tt.name, got)
		}
	}

	dep := byKey(mods, Mod{ModID: 900})
	if dep.State != StateDependency || dep.FileID != 9 || !slices.Contains(dep.IDs, "smapi:B.Dep") {
		t.Errorf("dependency = %+v", dep)
	}
	kinds := map[string]Problem{}
	for _, p := range probs {
		kinds[p.Kind+"|"+p.Name] = p
	}
	for _, want := range []string{ProblemRemoved + "|Nexus mod 500", ProblemNoFile + "|Nexus mod 800", ProblemBroken + "|Broken Mod", ProblemFree + "|"} {
		if _, ok := kinds[want]; !ok {
			t.Errorf("missing problem %q in %+v", want, probs)
		}
	}
	if p := kinds[ProblemRemoved+"|Nexus mod 500"]; p.URL != "https://www.nexusmods.com/stardewvalley/mods/500" {
		t.Errorf("removed problem has page %q", p.URL)
	}
	if p := kinds[ProblemBroken+"|Broken Mod"]; p.Key != byKey(mods, Mod{ModID: 1000}).Key || p.Detail != "1.6.15" {
		t.Errorf("broken problem = %+v", p)
	}
	for _, m := range mods {
		if m.Site == SiteNexus && m.ModID == 100 && m.SizeKB != 10 {
			t.Errorf("size = %d, want the Nexus size_kb", m.SizeKB)
		}
	}
}

func TestResolveSignedOutStaysWithTheLink(t *testing.T) {
	r := testResolver(nil, nil)
	r.signedIn = false
	mods, probs := r.resolve(t.Context(), []share.Ref{{ModID: 100, FileID: 1}, {ModID: 700, FileID: 7}})
	if mods[0].State != StateDownload || mods[0].FileID != 1 || mods[1].State != StateLater {
		t.Errorf("mods = %+v", mods)
	}
	for _, p := range probs {
		if p.Kind == ProblemRemoved || p.Kind == ProblemFree {
			t.Errorf("unexpected problem %+v", p)
		}
	}
}

func TestResolveFilesFailureIsUnconfirmed(t *testing.T) {
	r := testResolver(nil, nil)
	r.files = func(context.Context, nexus.Title, int) ([]nexus.File, error) { return nil, errors.New("offline") }
	mods, probs := r.resolve(t.Context(), []share.Ref{{ModID: 100, FileID: 1}})
	if mods[0].State != StateDownload || mods[0].FileID != 1 {
		t.Errorf("mods = %+v", mods)
	}
	if !slices.ContainsFunc(probs, func(p Problem) bool { return p.Kind == ProblemUnconfirmed }) {
		t.Errorf("problems = %+v", probs)
	}
}

func TestResolveGitHubInstalled(t *testing.T) {
	target := []profile.Entry{{Key: "gh", Source: profile.Source{Kind: profile.KindGitHub, Repo: "O/R", Tag: "v1", Asset: "a.zip"}}}
	r := testResolver(nil, target)
	mods, _ := r.resolve(t.Context(), []share.Ref{{GitHub: "o/r@v1/a.zip"}, {GitHub: "o/r@v2/a.zip"}})
	if mods[0].State != StateInstalled || mods[1].State != StateDownload || !mods[1].Unverified || mods[1].Tag != "v2" {
		t.Errorf("mods = %+v", mods)
	}
}
