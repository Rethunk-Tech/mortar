package devsvc

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// fixtureProject writes a filled template's files that Inspect and Edit read and change, and records the project.
func fixtureProject(t *testing.T, svc *Service) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "shop-drop")
	for name, body := range map[string]string{
		"src/ShopDrop/ShopDrop.csproj": `<Project><PropertyGroup><AssemblyName>com.ab.ShopDrop</AssemblyName><AssemblyTitle>ShopDrop</AssemblyTitle>` +
			`<RootNamespace>ShopDrop</RootNamespace><ThunderstoreName>ShopDrop</ThunderstoreName>` +
			`<ThunderstoreDescription>Lands &amp; drops.</ThunderstoreDescription><ThunderstoreCategories>mods;items</ThunderstoreCategories></PropertyGroup></Project>`,
		"src/ShopDrop/Plugin.cs": "namespace ShopDrop;\n// loads shopdrop.bundle\nclass ShopDropThing {}\n",
		"Directory.Build.props":  `<Project><PropertyGroup><ThunderstoreTeam>Ab</ThunderstoreTeam><Authors>A</Authors><Company>A</Company><Copyright>Copyright (c) 2026 A</Copyright></PropertyGroup></Project>`,
		"ShopDrop.slnx":          `<Solution><Project Path="src/ShopDrop/ShopDrop.csproj" /></Solution>`,
		"README.md":              "# ShopDrop\n",
		"LICENSE":                "MIT License\n\nCopyright (c) 2026 A\n",
	} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "artifacts", "bin", "ShopDrop", "debug")
	if err := svc.Settings.AddDevProject("lethal-company", settings.DevProject{Dir: dir, Name: "ShopDrop", Profile: "p", Output: out}); err != nil {
		t.Fatal(err)
	}
	return dir
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInspectReadsTheProjectFiles(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	info, err := svc.Inspect(t.Context(), "lethal-company", dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "ShopDrop" || info.GUID != "com.ab.ShopDrop" || info.IDPrefix != "com.ab" || info.Author != "A" ||
		info.Team != "Ab" || info.Description != "Lands & drops." || strings.Join(info.Categories, ",") != "mods,items" || info.Missing ||
		info.Icon != "" {
		t.Fatalf("info = %+v", info)
	}
	// With an icon.png the card gets it as a data URL.
	var icon bytes.Buffer
	if err := png.Encode(&icon, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, "icon.png"), icon.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if info, err = svc.Inspect(t.Context(), "lethal-company", dir); err != nil || !strings.HasPrefix(info.Icon, "data:image/png;base64,") {
		t.Fatalf("icon = %.40q, %v", info.Icon, err)
	}
}

func TestEditRenamesAndRewritesTheProject(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	info, err := svc.Edit(t.Context(), "lethal-company", dir, ProjectEdit{
		Name: "LandDrop", Description: "Lands <anywhere>.", Author: "B & C", Team: "Bc", IDPrefix: "com.bc", Categories: []string{"misc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "LandDrop" || info.GUID != "com.bc.LandDrop" || info.Author != "B & C" || info.Description != "Lands <anywhere>." {
		t.Errorf("info = %+v", info)
	}
	csproj := read(t, filepath.Join(dir, "src", "LandDrop", "LandDrop.csproj"))
	if !strings.Contains(csproj, "<ThunderstoreDescription>Lands &lt;anywhere&gt;.</ThunderstoreDescription>") || !strings.Contains(csproj, "<RootNamespace>LandDrop</RootNamespace>") {
		t.Errorf("csproj = %s", csproj)
	}
	if code := read(t, filepath.Join(dir, "src", "LandDrop", "Plugin.cs")); code != "namespace LandDrop;\n// loads landdrop.bundle\nclass ShopDropThing {}\n" {
		t.Errorf("code = %q", code)
	}
	if slnx := read(t, filepath.Join(dir, "LandDrop.slnx")); !strings.Contains(slnx, "src/LandDrop/LandDrop.csproj") {
		t.Errorf("slnx = %s", slnx)
	}
	if lic := read(t, filepath.Join(dir, "LICENSE")); !strings.Contains(lic, "2026 B & C") {
		t.Errorf("license = %s", lic)
	}
	if props := read(t, filepath.Join(dir, "Directory.Build.props")); !strings.Contains(props, "Copyright (c) 2026 B &amp; C</Copyright>") {
		t.Errorf("props = %s", props)
	}
	gp := svc.Settings.Get().GamePrefs("lethal-company")
	want := filepath.Join(dir, "artifacts", "bin", "LandDrop", "debug")
	if gp.DevProjects[0].Name != "LandDrop" || gp.DevProjects[0].Output != want || gp.ExtraModsProfile(want) != "p" || len(gp.ExtraModsFolderList()) != 1 {
		t.Errorf("record %+v, folders %v", gp.DevProjects, gp.ExtraModsFolderList())
	}
}

func TestInspectFollowsARenameMadeOutsideAndSeesAMissingProject(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	if err := os.Rename(filepath.Join(dir, "src", "ShopDrop"), filepath.Join(dir, "src", "Other")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "src", "Other", "ShopDrop.csproj"), filepath.Join(dir, "src", "Other", "Other.csproj")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Inspect(t.Context(), "lethal-company", dir); err != nil {
		t.Fatal(err)
	}
	if p := svc.Projects("lethal-company")[0]; p.Name != "Other" || !strings.HasSuffix(p.Output, filepath.Join("bin", "Other", "debug")) {
		t.Errorf("record = %+v", p)
	}
	if err := fsx.RemoveAll(filepath.Join(dir, "src")); err != nil {
		t.Fatal(err)
	}
	if info, err := svc.Inspect(t.Context(), "lethal-company", dir); err != nil || !info.Missing {
		t.Errorf("missing = %+v, %v", info, err)
	}
}

func TestEditRefusesBadDetails(t *testing.T) {
	ok := ProjectEdit{Name: "ShopDrop", Description: "d", Author: "A", Team: "Ab", IDPrefix: "com.ab", Categories: []string{"mods"}}
	if err := ok.check(); err != nil {
		t.Fatal(err)
	}
	for _, m := range []func(*ProjectEdit){
		func(e *ProjectEdit) { e.Name = "shop" },
		func(e *ProjectEdit) { e.Description = "a\nb" },
		func(e *ProjectEdit) { e.Author = " " },
		func(e *ProjectEdit) { e.Team = "a b" },
		func(e *ProjectEdit) { e.IDPrefix = "Com" },
		func(e *ProjectEdit) { e.Categories = []string{"Items & More"} },
	} {
		bad := ok
		m(&bad)
		if bad.check() == nil {
			t.Errorf("%+v was accepted", bad)
		}
	}
}
