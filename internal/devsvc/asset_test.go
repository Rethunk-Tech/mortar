package devsvc

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
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
	if body, err := fsx.ReadFile(filepath.Join(dir, "sound_pack.json")); err != nil || !strings.Contains(string(body), `"name": "Quacks"`) {
		t.Errorf("starter file = %s (%v)", body, err)
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
	if err := os.WriteFile(filepath.Join(dir, "sounds", "quack.ogg"), []byte("ogg"), 0o600); err != nil {
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
	want := []string{"CHANGELOG.md", "LICENSE", "README.md", "icon.png", "manifest.json", "replacers/README.txt", "sound_pack.json", "sounds/README.txt", "sounds/quack.ogg"}
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
