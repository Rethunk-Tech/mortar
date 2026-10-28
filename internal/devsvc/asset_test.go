package devsvc

import (
	"archive/zip"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestAnAssetProjectIsItsOwnPackage(t *testing.T) {
	svc, d := testService(t, testTemplate)
	parent := t.TempDir()
	p, err := svc.Create(t.Context(), NewMod{
		Game: "lethal-company", Kind: "sounds", Name: "Quacks", Parent: parent, Repo: "quacks",
		Values: map[string]string{"description": "Ducks.", "modAuthor": "Me", "team": "Ab"},
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(parent, "quacks")
	if slices.ContainsFunc(d.calls, func(c []string) bool { return c[0] == "dotnet" }) {
		t.Errorf("an asset project ran dotnet: %v", d.calls)
	}
	if p.Kind != "sounds" || p.Output != dir || p.Untrusted || p.TemplateIcon == "" {
		t.Fatalf("project %+v", p)
	}
	if body, err := fsx.ReadFile(filepath.Join(dir, "plugins", "sound_pack.json")); err != nil || !strings.Contains(string(body), `"name": "Quacks"`) {
		t.Errorf("starter file = %s (%v)", body, err)
	}
	var watched []string
	for _, f := range AssetSubfolders(svc.Settings.Get(), []string{"lethal-company"}) {
		if f.Root == dir {
			watched = append(watched, filepath.Base(f.Dir))
		}
	}
	if slices.Sort(watched); !slices.Equal(watched, []string{"plugins", "replacers", "sounds"}) {
		t.Errorf("watched subfolders %v", watched)
	}
	if err := svc.AddDependency("lethal-company", dir, "loaforc-loaforcsSoundAPI", "2.0.13", true); err != nil {
		t.Fatal(err)
	}
	if res, err := svc.Build(t.Context(), "lethal-company", dir); err != nil || !res.OK {
		t.Errorf("build = %+v, %v", res, err)
	}
	info, err := svc.Inspect(t.Context(), "lethal-company", dir)
	if err != nil || info.Kind != "sounds" || info.Package != "Ab-Quacks" || info.Missing {
		t.Errorf("info = %+v, %v", info, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins", "sounds", "quack.ogg"), []byte("ogg"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := svc.Pack(t.Context(), "lethal-company", dir)
	if err != nil || !r.OK || !slices.Equal(r.Issues, []string{"icon.png is still the placeholder; draw the mod its own"}) {
		t.Fatalf("pack = %+v, %v", r, err)
	}
	zr, err := zip.OpenReader(r.Zip)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	slices.Sort(names)
	want := []string{"CHANGELOG.md", "LICENSE", "README.md", "icon.png", "manifest.json", "plugins/replacers/README.txt", "plugins/sound_pack.json", "plugins/sounds/README.txt", "plugins/sounds/quack.ogg"}
	if !slices.Equal(names, want) || filepath.Base(r.Zip) != "Ab-Quacks-0.1.0.zip" {
		t.Errorf("zip %s holds %v", r.Zip, names)
	}
	manifest := zipText(t, zr, "manifest.json")
	if !strings.Contains(manifest, `"loaforc-loaforcsSoundAPI-2.0.13"`) {
		t.Errorf("manifest = %s", manifest)
	}
	if _, err := svc.Create(t.Context(), NewMod{Game: "lethal-company", Kind: "no-such-kind", Name: "Art", Parent: parent, Repo: "art"}); err == nil {
		t.Error("a kind the catalog does not offer was made")
	}
}

func zipText(t *testing.T, zr *zip.ReadCloser, name string) string {
	t.Helper()
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = rc.Close() }()
			body, err := io.ReadAll(rc)
			if err != nil {
				t.Fatal(err)
			}
			return string(body)
		}
	}
	t.Fatalf("%s is not in the zip", name)
	return ""
}

// A kind's folder has to reach the profile as it is: BepInEx keeps subfolders only below plugins/ (or config/ and the
// like) and flattens the rest, so a folder at the package's top would never be found by its loader.
func TestEveryAssetKindKeepsItsFolderWhenInstalled(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	kinds := svc.AssetKinds("lethal-company")
	if len(kinds) == 0 {
		t.Fatal("the catalog offers no asset kinds")
	}
	for _, k := range kinds {
		for _, f := range append(slices.Collect(maps.Keys(k.Files)), k.Folder+"/x.png") {
			if dest := bepinex5.Route(f, "Ab-Pack", nil); dest != path.Join("BepInEx/plugins/Ab-Pack", strings.TrimPrefix(f, "plugins/")) {
				t.Errorf("%s: %s installs at %s", k.ID, f, dest)
			}
		}
	}
}

func assetDir(t *testing.T, record string) string {
	t.Helper()
	dir := t.TempDir()
	if err := writeAll(filepath.Join(dir, ".mortar", assetRecordFile), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAnAddedAssetProjectInstallsOnlyOnceTrusted(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	const game = "lethal-company"
	dir := assetDir(t, `{"kind":"sounds","name":"Quacks","team":"Ab"}`)
	p, err := svc.Add(t.Context(), game, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Untrusted || p.Output == dir || !slices.Contains(svc.Settings.Get().GamePrefs(game).ExtraModsFolderList(), p.Output) {
		t.Fatalf("added = %+v", p)
	}
	if err := svc.Trust(game, dir); err != nil {
		t.Fatal(err)
	}
	got := svc.Projects(game)[0]
	if got.Untrusted || got.Output != dir || !slices.Contains(svc.Settings.Get().GamePrefs(game).ExtraModsFolderList(), dir) || slices.Contains(svc.Settings.Get().GamePrefs(game).ExtraModsFolderList(), p.Output) {
		t.Errorf("trusted = %+v", got)
	}
}

func TestAnAssetRecordCannotNameAPathOutsideTheProject(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	const game = "lethal-company"
	if _, err := svc.Add(t.Context(), game, assetDir(t, `{"kind":"sounds","name":"Quacks","team":"../../x"}`), ""); err == nil {
		t.Error("a team with path parts was accepted")
	}
	dir := assetDir(t, `{"kind":"sounds","name":"Quacks","team":"Ab"}`)
	if err := writeAll(filepath.Join(dir, "manifest.json"), []byte(`{"version_number":"../../1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := svc.Add(t.Context(), game, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Trust(game, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Pack(t.Context(), game, p.Dir); err == nil {
		t.Error("a manifest version with path parts was packed")
	}
}

func TestAnAssetProjectUploadsUnderItsRecordsTeam(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	const game = "lethal-company"
	dir := assetDir(t, `{"kind":"sounds","name":"Quacks","team":"Ab"}`)
	if _, err := svc.Add(t.Context(), game, dir, ""); err != nil {
		t.Fatal(err)
	}
	if team, err := svc.projectTeam(game, dir); err != nil || team != "Ab" {
		t.Errorf("team = %q, %v", team, err)
	}
}

func TestAnAssetProjectReleasesHereAndEditsItsDetails(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = nil
	const game = "lethal-company"
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	git := func(dir string, args ...string) {
		t.Helper()
		if _, err := svc.run(t.Context(), dir, "git", args...); err != nil {
			t.Fatal(err)
		}
	}
	dir := assetDir(t, `{"kind":"sounds","name":"Quacks","author":"A","team":"Ab"}`)
	if err := writeAll(filepath.Join(dir, "manifest.json"), []byte(`{"name":"Quacks","version_number":"0.1.0","description":"d","dependencies":["BepInEx-BepInExPack-5.4.2100"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	hub := filepath.Join(t.TempDir(), "github.com", "o", "quacks.git")
	if err := os.MkdirAll(hub, 0o750); err != nil {
		t.Fatal(err)
	}
	git(hub, "init", "-q", "--bare")
	git(dir, "init", "-q")
	git(dir, "add", ".")
	git(dir, "commit", "-q", "-m", "start")
	git(dir, "remote", "add", "origin", hub)
	if _, err := svc.Add(t.Context(), game, dir, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Release(t.Context(), game, dir, "0.1.0", []string{"First"}); usererr.KindOf(err) != usererr.Unconfirmed {
		t.Fatalf("untrusted release = %v", err)
	}
	if err := svc.Trust(game, dir); err != nil {
		t.Fatal(err)
	}
	if v, err := svc.NextVersions(t.Context(), game, dir); err != nil || v.GitHub != "" {
		t.Fatalf("versions = %+v, %v: no workflow publishes an asset project", v, err)
	}
	if remote, err := svc.Release(t.Context(), game, dir, "0.1.0", []string{"First"}); err != nil || remote != "" {
		t.Fatalf("release = %q, %v", remote, err)
	}
	git(hub, "rev-parse", "-q", "--verify", "refs/tags/v0.1.0")

	if err := svc.AddDependency(game, dir, "Ab-Other", "1.0.0", false); err != nil {
		t.Fatal(err)
	}
	if deps, err := svc.Dependencies(game, dir); err != nil || len(deps) != 1 || deps[0].Package != "Ab-Other" || deps[0].Version != "1.0.0" {
		t.Fatalf("dependencies = %+v, %v", deps, err)
	}
	if err := svc.RemoveDependency(game, dir, "Ab-Other"); err != nil {
		t.Fatal(err)
	}
	if deps, _ := svc.Dependencies(game, dir); len(deps) != 0 {
		t.Errorf("dependencies = %+v", deps)
	}
	info, err := svc.Edit(t.Context(), game, dir, ProjectEdit{Name: "Quacks", Description: "Loud.", Author: "B", Team: "Bc"})
	if err != nil || info.Description != "Loud." || info.Author != "B" || info.Team != "Bc" {
		t.Fatalf("edited = %+v, %v", info, err)
	}
	if _, err := svc.Edit(t.Context(), game, dir, ProjectEdit{Name: "Other", Description: "d", Author: "B", Team: "Bc"}); err == nil {
		t.Error("a rename of a project of files was accepted")
	}
	if pkgs := svc.Packages(game); !slices.Equal(pkgs, []string{"Bc-Quacks"}) {
		t.Errorf("packages = %v", pkgs)
	}
}

func TestMoveRepointsAProjectAtAnotherProfile(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	const game = "lethal-company"
	dir := assetDir(t, `{"kind":"sounds","name":"Quacks","team":"Ab"}`)
	p, err := svc.Add(t.Context(), game, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	other, err := svc.Profiles.Create(game, "Other")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Move(game, dir, "missing"); err == nil {
		t.Error("a profile that does not exist was accepted")
	}
	if err := svc.Move(game, dir, other.ID); err != nil {
		t.Fatal(err)
	}
	gp := svc.Settings.Get().GamePrefs(game)
	if gp.DevProjects[0].Profile != other.ID || gp.ExtraModsProfile(p.Output) != other.ID {
		t.Errorf("project %+v, folder profile %q", gp.DevProjects[0], gp.ExtraModsProfile(p.Output))
	}
}

func TestCreateAssetRefusesATeamTheRecordWouldRejectBeforeWritingAnything(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	parent := t.TempDir()
	_, err := svc.Create(t.Context(), NewMod{
		Game: "lethal-company", Kind: "sounds", Name: "Quacks", Parent: parent, Repo: "quacks",
		Values: map[string]string{"description": "Ducks.", "modAuthor": "Me", "team": "_Ab"},
	})
	if usererr.KindOf(err) != usererr.Invalid {
		t.Fatalf("err = %v", err)
	}
	if entries, _ := os.ReadDir(parent); len(entries) != 0 {
		t.Errorf("a refused project left %v", entries)
	}
}

func TestAssetParamsHaveGUIText(t *testing.T) {
	src, err := os.ReadFile("../../frontend/src/develop/ownParams.ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range assetParams() {
		if !p.Own || !strings.Contains(string(src), "\n  "+p.ID+": {") {
			t.Errorf("param %q is not marked own or has no entry in ownParams.ts", p.ID)
		}
	}
}
