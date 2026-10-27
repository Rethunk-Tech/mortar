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
	if _, err := svc.Create(t.Context(), NewMod{Game: "lethal-company", Kind: "posters", Name: "Art", Parent: parent, Repo: "art"}); err == nil {
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
