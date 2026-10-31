package devsvc

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

const contentGame = "lethal-company"

func TestDetectFindsContentInTheNewAndOldLayouts(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	cases := []struct {
		name            string
		files           map[string]string
		kind            string
		content, legacy []string
	}{
		{"new layout", map[string]string{"plugins/mortar/tv/a.mp4": "v", "plugins/mortar/posters/a.png": "p"}, kindContent, []string{"posters", "tv"}, []string{}},
		{"old layout", map[string]string{"plugins/LethalPosters/tips/a.png": "p", "plugins/Custom Songs/a.ogg": "s"}, kindContent, []string{}, []string{"tips", "boombox"}},
		{"package top", map[string]string{"mortar/tv/a.mp4": "v"}, kindContent, []string{}, []string{"tv"}},
		{"both", map[string]string{"plugins/mortar/paintings/a.png": "p", "plugins/Television Videos/a.webm": "v"}, kindContent, []string{"paintings"}, []string{"tv"}},
		{"code project", map[string]string{"Pack.csproj": "<Project/>", "content/mortar/boombox/a.ogg": "s"}, kindCode, []string{"boombox"}, []string{}},
		{"neither", map[string]string{"plugins/sounds/quack.ogg": "o"}, "sounds", []string{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, err := svc.Detect(contentGame, adoptFolder(t, "Pack", c.files))
			if err != nil || a.Kind != c.kind || !slices.Equal(a.Content, c.content) || !slices.Equal(a.Legacy, c.legacy) {
				t.Fatalf("detect = %+v, %v", a, err)
			}
			if c.kind != kindCode && !slices.Contains(a.Kinds, kindContent) {
				t.Errorf("kinds %v leave out the content pack", a.Kinds)
			}
		})
	}
	kinds := svc.AssetKinds(contentGame)
	if slices.ContainsFunc(kinds, func(k components.AssetKind) bool { return k.ID == "posters" || k.ID == "television" }) {
		t.Errorf("a kind the content pack supersedes is still offered: %v", kinds)
	}
}

func TestMovingOldLayoutContentKeepsWhatIsAlreadyMoved(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := adoptFolder(t, "Pack", map[string]string{
		"plugins/LethalPosters/posters/a.png": "old",
		"plugins/LethalPosters/posters/b.png": "b",
		"plugins/mortar/posters/a.png":        "new",
		"plugins/Custom Songs/s.ogg":          "s",
		"mortar/tv/v.webm":                    "v",
	})
	moved, err := svc.MoveLegacyContent(contentGame, dir)
	if err != nil || moved != 3 {
		t.Fatalf("moved %d, %v", moved, err)
	}
	for f, want := range map[string]string{
		"plugins/mortar/posters/a.png":        "new",
		"plugins/mortar/posters/b.png":        "b",
		"plugins/mortar/boombox/s.ogg":        "s",
		"plugins/mortar/tv/v.webm":            "v",
		"plugins/LethalPosters/posters/a.png": "old",
	} {
		if body, err := safefs.ReadFile(filepath.Join(dir, filepath.FromSlash(f))); err != nil || string(body) != want {
			t.Errorf("%s = %q, %v", f, body, err)
		}
	}
	if isDir(filepath.Join(dir, "plugins", "Custom Songs")) || isDir(filepath.Join(dir, "mortar")) {
		t.Error("an emptied old folder stayed")
	}
}

func TestA038RecordOfASupersededKindOpensAsTheContentPack(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	for _, old := range []string{"posters", "paintings", "television", "boombox"} {
		t.Run(old, func(t *testing.T) {
			dir := assetDir(t, `{"kind":"`+old+`","name":"Pack","team":"Ab"}`)
			p, err := svc.Add(t.Context(), contentGame, dir, "")
			if err != nil || p.Kind != kindContent {
				t.Fatalf("added %+v, %v", p, err)
			}
			if a, err := svc.Detect(contentGame, dir); err != nil || !a.Known || a.Kind != kindContent {
				t.Errorf("detect = %+v, %v", a, err)
			}
		})
	}
	dir := assetDir(t, `{"kind":"sounds","name":"Quacks","team":"Ab"}`)
	if p, err := svc.Add(t.Context(), contentGame, dir, ""); err != nil || p.Kind != "sounds" {
		t.Errorf("sounds added as %+v, %v", p, err)
	}
}

func TestReleaseRefusesAnEmptyContentFolder(t *testing.T) {
	folders := contentFolders(contentGame)
	cases := []struct {
		name  string
		files map[string]string
		root  string
		pack  bool
		want  []string
	}{
		{"filled", map[string]string{"plugins/mortar/tv/a.mp4": "v"}, packContentRoot, true, []string{}},
		{"empty kind", map[string]string{"plugins/mortar/tv/README.txt": "r", "plugins/mortar/posters/a.png": "p"}, packContentRoot, true, []string{"plugins/mortar/tv/ holds no *.mp4, *.webm file"}},
		{"package top", map[string]string{"mortar/tv/a.mp4": "v", "plugins/mortar/tv/a.mp4": "v"}, packContentRoot, true, []string{"mortar/tv/ installs flattened from the package's top; move it under plugins/mortar/"}},
		{"no kind", map[string]string{"README.md": "r"}, packContentRoot, true, []string{"the content pack has no content folder; add one under plugins/mortar/"}},
		{"old layout counts", map[string]string{"plugins/Custom Songs/a.ogg": "s"}, packContentRoot, true, []string{}},
		{"code without content", map[string]string{"Pack.csproj": "<Project/>"}, codeContentRoot, false, []string{}},
		{"code with an empty kind", map[string]string{"content/mortar/boombox/README.txt": "r"}, codeContentRoot, false, []string{"content/mortar/boombox/ holds no *.mp3, *.ogg, *.wav file"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := contentIssues(adoptFolder(t, "Pack", c.files), c.root, folders, c.pack); !slices.Equal(got, c.want) {
				t.Errorf("issues = %q, want %q", got, c.want)
			}
		})
	}
}

func TestANewContentPackGetsATickedFolderEach(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	parent := t.TempDir()
	values := map[string]string{"description": "Art.", "modAuthor": "Me", "team": "Ab"}
	if _, err := svc.Create(t.Context(), NewMod{Game: contentGame, Kind: kindContent, Name: "Art", Parent: parent, Repo: "none", Values: values}); err == nil {
		t.Error("a content pack with nothing ticked was made")
	}
	values[contentParam] = "posters|tv"
	p, err := svc.Create(t.Context(), NewMod{Game: contentGame, Kind: kindContent, Name: "Art", Parent: parent, Repo: "art", Values: values})
	if err != nil || p.Kind != kindContent {
		t.Fatalf("created %+v, %v", p, err)
	}
	found, _ := detectContent(p.Dir, contentFolders(contentGame))
	if !slices.Equal(found, []string{"posters", "tv"}) {
		t.Errorf("folders %v", found)
	}
	form, err := svc.AssetForm(contentGame, kindContent)
	if err != nil || !slices.ContainsFunc(form.Parameters, func(p Param) bool { return p.ID == contentParam && p.Required }) {
		t.Errorf("form = %+v, %v", form, err)
	}
}
