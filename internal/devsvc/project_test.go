package devsvc

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/msbuild"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
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

func TestInspectReadsTheProjectFiles(t *testing.T) {
	svc, run := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	info, err := svc.Inspect(t.Context(), "lethal-company", dir)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(run.calls, func(c []string) bool { return len(c) > 1 && c[0] == "git" && c[1] == "log" }) {
		t.Errorf("a card refresh read the whole history: %v", run.calls)
	}
	if info.Name != "ShopDrop" || info.GUID != "com.ab.ShopDrop" || info.IDPrefix != "com.ab" || info.Author != "A" ||
		info.Team != "Ab" || info.Description != "Lands & drops." || strings.Join(info.Categories, ",") != "mods,items" || info.Missing ||
		info.Icon != "" {
		t.Fatalf("info = %+v", info)
	}
	// A mod only its own player needs tests in one copy.
	rec := filepath.Join(dir, ".mortar", "template.json")
	if err := os.MkdirAll(filepath.Dir(rec), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rec, []byte(`{"name":"ShopDrop","values":{"audience":"client"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if info, _ = svc.Inspect(t.Context(), "lethal-company", dir); !info.ClientOnly {
		t.Fatal("a client-only project is not ClientOnly")
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
	if info.BuiltAt != 0 || info.Stale {
		t.Fatalf("unbuilt: %d %v", info.BuiltAt, info.Stale)
	}
	// A build found on disk counts, and a source written after it makes it stale.
	p := svc.Settings.Get().GamePrefs("lethal-company").DevProjects[0]
	dll := filepath.Join(p.Output, "com.ab.ShopDrop.dll")
	if err := os.MkdirAll(p.Output, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(dll, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	for _, f := range []string{"ShopDrop.csproj", "Plugin.cs"} {
		if err := os.Chtimes(filepath.Join(dir, "src", "ShopDrop", f), old, old); err != nil {
			t.Fatal(err)
		}
	}
	if info, _ = svc.Inspect(t.Context(), "lethal-company", dir); info.BuiltAt == 0 || info.Stale {
		t.Fatalf("current build: %d %v", info.BuiltAt, info.Stale)
	}
	if err := os.Chtimes(dll, old.Add(-time.Hour), old.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if info, _ = svc.Inspect(t.Context(), "lethal-company", dir); !info.Stale {
		t.Fatal("a source newer than the build is not stale")
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
	csproj := testfs.Read(t, filepath.Join(dir, "src", "LandDrop", "LandDrop.csproj"))
	if !strings.Contains(csproj, "<ThunderstoreDescription>Lands &lt;anywhere&gt;.</ThunderstoreDescription>") || !strings.Contains(csproj, "<RootNamespace>LandDrop</RootNamespace>") {
		t.Errorf("csproj = %s", csproj)
	}
	if code := testfs.Read(t, filepath.Join(dir, "src", "LandDrop", "Plugin.cs")); code != "namespace LandDrop;\n// loads landdrop.bundle\nclass ShopDropThing {}\n" {
		t.Errorf("code = %q", code)
	}
	if slnx := testfs.Read(t, filepath.Join(dir, "LandDrop.slnx")); !strings.Contains(slnx, "src/LandDrop/LandDrop.csproj") {
		t.Errorf("slnx = %s", slnx)
	}
	if lic := testfs.Read(t, filepath.Join(dir, "LICENSE")); !strings.Contains(lic, "2026 B & C") {
		t.Errorf("license = %s", lic)
	}
	if props := testfs.Read(t, filepath.Join(dir, "Directory.Build.props")); !strings.Contains(props, "Copyright (c) 2026 B &amp; C</Copyright>") {
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
		func(e *ProjectEdit) { e.Team = "_Ab" },
		func(e *ProjectEdit) { e.Team = "Ab_" },
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

func TestSetPropEscapesWhatMSBuildExpands(t *testing.T) {
	const text = "Costs $(Version) 100%3B more @(Compile); it's <b> *?"
	body := "<Project><PropertyGroup><Description>x</Description></PropertyGroup></Project>"
	out := setProp(body, "Description", text)
	if strings.Contains(out, "$(Version)") || strings.Contains(out, "@(Compile)") {
		t.Fatalf("MSBuild would expand the value: %s", out)
	}
	if got := msbuild.Property(out, "Description"); got != text {
		t.Fatalf("round trip = %q, want %q", got, text)
	}
}

// MSBuild splits a string[] property on ;, so the categories keep their separator.
func TestEditWritesCategoriesAsAList(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	info, err := svc.Edit(t.Context(), "lethal-company", dir, ProjectEdit{
		Name: "ShopDrop", Description: "d", Author: "A", Team: "Ab", IDPrefix: "com.ab", Categories: []string{"mods", "bepinex"},
	})
	if err != nil {
		t.Fatal(err)
	}
	csproj := testfs.Read(t, filepath.Join(dir, "src", "ShopDrop", "ShopDrop.csproj"))
	if !strings.Contains(csproj, "<ThunderstoreCategories>mods;bepinex</ThunderstoreCategories>") || strings.Join(info.Categories, ",") != "mods,bepinex" {
		t.Errorf("categories %v in %s", info.Categories, csproj)
	}
}

func TestEditMatchesAndWritesTheAuthorEscapedLikeTheTemplate(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	propsPath := filepath.Join(dir, "Directory.Build.props")
	if err := fsx.WriteFile(propsPath, []byte(`<Project><PropertyGroup><ThunderstoreTeam>Ab</ThunderstoreTeam><Authors>O%27Brien</Authors><Company>O%27Brien</Company><Copyright>Copyright (c) 2026 O%27Brien</Copyright></PropertyGroup></Project>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Edit(t.Context(), "lethal-company", dir, ProjectEdit{
		Name: "ShopDrop", Description: "d", Author: "$(Version)", Team: "Ab", IDPrefix: "com.ab", Categories: []string{"mods"},
	}); err != nil {
		t.Fatal(err)
	}
	if props := testfs.Read(t, propsPath); !strings.Contains(props, "Copyright (c) 2026 %24(Version)</Copyright>") {
		t.Errorf("props = %s", props)
	}
}

func TestEditMovesTheTeamInDevelopingMd(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	doc := filepath.Join(dir, "DEVELOPING.md")
	if err := fsx.WriteFile(doc, []byte("- Linux: `secret-tool store --label \"Thunderstore Ab\" service thunderstore username Ab`\n  `New-StoredCredential -Target 'thunderstore:Ab' -Password <token>`\n- Other: username Abc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Edit(t.Context(), "lethal-company", dir, ProjectEdit{
		Name: "ShopDrop", Description: "d", Author: "A", Team: "Bc", IDPrefix: "com.ab", Categories: []string{"mods"},
	}); err != nil {
		t.Fatal(err)
	}
	// "username Abc" names another team and stays.
	if got := testfs.Read(t, doc); strings.Contains(got, "username Ab`") || !strings.Contains(got, "username Bc`") ||
		!strings.Contains(got, "thunderstore:Bc'") || !strings.Contains(got, "username Abc") || !strings.Contains(got, "Thunderstore Bc") {
		t.Errorf("DEVELOPING.md = %s", got)
	}
}

func TestInspectRunsNoGitForAnUntrustedProject(t *testing.T) {
	svc, run := testService(t, testTemplate)
	dir := fixtureProject(t, svc)
	out := filepath.Join(dir, "artifacts", "bin", "ShopDrop", "debug")
	if err := svc.Settings.AddDevProject("lethal-company", settings.DevProject{Dir: dir, Name: "ShopDrop", Profile: "p", Output: out, Untrusted: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Inspect(t.Context(), "lethal-company", dir); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.NextVersions(t.Context(), "lethal-company", dir); err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(run.calls, func(c []string) bool { return c[0] == "git" }) {
		t.Errorf("git ran in a project the user has not trusted: %v", run.calls)
	}
}
