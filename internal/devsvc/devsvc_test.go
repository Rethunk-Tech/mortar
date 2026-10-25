package devsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func templateZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"tpl-v1/ModTemplate.slnx":                   `<Project Path="src/ModTemplate/ModTemplate.csproj" />`,
		"tpl-v1/src/ModTemplate/ModTemplate.csproj": `<AssemblyName>modauthor.ModTemplate</AssemblyName><D>ModDescription</D><U>ModRepositoryUrl</U>`,
		"tpl-v1/LICENSE":                            "Copyright (c) ModYear ModAuthor",
		"tpl-v1/Directory.Build.props":              "<ThunderstoreTeam>ModTeam</ThunderstoreTeam>",
		"tpl-v1/icon.png":                           "\x89PNG\x00ModTemplate",
		"tpl-v1/Config.Build.user.props.template":   "<GameRootDir />",
	} {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCreateFromTheArchiveFillsTheTemplateAndWiresTheProfile(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	body := templateZip(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	t.Cleanup(srv.Close)
	_, profiles := testenv.Stores(t)
	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{
		Home:       t.TempDir(),
		Settings:   settings.NewService(store),
		Profiles:   profile.NewService(profiles, t.TempDir(), store),
		ArchiveURL: func(components.DevTemplate) string { return srv.URL },
		Run: func(context.Context, string, string, ...string) ([]byte, error) {
			return nil, errors.New("no tools here")
		},
	}
	parent := t.TempDir()
	p, err := svc.Create(t.Context(), NewMod{
		Game: "lethal-company", Name: "ShopDrop", Author: "A & B", Team: "Ab", GUIDPrefix: "com.ab",
		Description: "Lands <somewhere> new.", Parent: parent, Repo: "shop-drop",
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(parent, "shop-drop")
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if got := read("src/ShopDrop/ShopDrop.csproj"); got != `<AssemblyName>com.ab.ShopDrop</AssemblyName><D>Lands &lt;somewhere&gt; new.</D><U></U>` {
		t.Errorf("csproj = %s", got)
	}
	if got := read("LICENSE"); !strings.HasSuffix(got, " A &amp; B") || strings.Contains(got, "ModYear") {
		t.Errorf("license = %s", got)
	}
	if got := read("Directory.Build.props"); got != "<ThunderstoreTeam>Ab</ThunderstoreTeam>" {
		t.Errorf("props = %s", got)
	}
	if got := read("icon.png"); !strings.Contains(got, "ModTemplate") {
		t.Error("a binary file was rewritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "ShopDrop.slnx")); err != nil {
		t.Error(err)
	}
	gp := svc.Settings.Get().GamePrefs("lethal-company")
	if len(gp.DevProjects) != 1 || gp.DevProjects[0] != p || p.Profile == "" || gp.ExtraModsProfile(p.Output) != p.Profile {
		t.Fatalf("project %+v, settings %+v", p, gp.DevProjects)
	}
	if fi, err := os.Stat(p.Output); err != nil || !fi.IsDir() {
		t.Errorf("build output folder: %v", err)
	}
	if _, err := svc.Create(t.Context(), NewMod{
		Game: "lethal-company", Name: "ShopDrop", Author: "A", Team: "Ab", GUIDPrefix: "com.ab",
		Description: "d", Parent: parent, Repo: "shop-drop",
	}); err == nil {
		t.Error("a second project in the same folder was made")
	}
	if err := svc.Forget("lethal-company", dir); err != nil || len(svc.Projects("lethal-company")) != 0 ||
		len(svc.Settings.Get().GamePrefs("lethal-company").ExtraModsFolderList()) != 0 {
		t.Errorf("forget left %+v (%v)", svc.Settings.Get().GamePrefs("lethal-company"), err)
	}
}

func TestNewModRefusesUnsafeNames(t *testing.T) {
	ok := NewMod{Name: "ShopDrop", Author: "A", Team: "Ab", GUIDPrefix: "com.ab", Description: "d", Parent: "/p", Repo: "r"}
	if err := ok.check(); err != nil {
		t.Fatal(err)
	}
	for _, m := range []func(*NewMod){
		func(m *NewMod) { m.Name = "shop drop" },
		func(m *NewMod) { m.Repo = ".." },
		func(m *NewMod) { m.Repo = "a/b" },
		func(m *NewMod) { m.Repo = "-c" },
		func(m *NewMod) { m.Parent = "rel" },
		func(m *NewMod) { m.GUIDPrefix = "Com.AB" },
		func(m *NewMod) { m.Team = "a b" },
		func(m *NewMod) { m.GitHub, m.Owner = true, "--help" },
		func(m *NewMod) { m.Description = strings.Repeat("x", maxDescription+1) },
	} {
		bad := ok
		m(&bad)
		if bad.check() == nil {
			t.Errorf("%+v was accepted", bad)
		}
	}
}

func TestParseProblemsReadsMSBuildErrorsOnce(t *testing.T) {
	out := `  Plugin.cs(12,5): error CS1002: ; expected [/p/src/A/A.csproj]
/p/src/A/Plugin.cs(3,1,3,9): warning CS0168: The variable 'x' is declared but never used [/p/src/A/A.csproj]
  Plugin.cs(12,5): error CS1002: ; expected [/p/src/A/A.csproj]
Build FAILED.
`
	got := ParseProblems(out)
	want := []Problem{
		{File: "Plugin.cs", Line: 12, Column: 5, Severity: "error", Code: "CS1002", Message: "; expected"},
		{File: "/p/src/A/Plugin.cs", Line: 3, Column: 1, Severity: "warning", Code: "CS0168", Message: "The variable 'x' is declared but never used"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("problems = %+v", got)
	}
}

func TestNextVersionsAndChangelogEntries(t *testing.T) {
	if v := nextVersions(""); v != (Versions{Patch: "0.1.0", Minor: "0.1.0", Major: "0.1.0"}) {
		t.Errorf("first = %+v", v)
	}
	if v := nextVersions("1.4.9"); v != (Versions{Current: "1.4.9", Patch: "1.4.10", Minor: "1.5.0", Major: "2.0.0"}) {
		t.Errorf("after 1.4.9 = %+v", v)
	}
	log := "# Changelog\n\n## 0.1.0\n\n- First release.\n"
	got := AddChangelogEntry(log, "0.2.0", []string{" Lands anywhere. ", "Fixes a crash."})
	if want := "# Changelog\n\n## 0.2.0\n\n- Lands anywhere.\n- Fixes a crash.\n\n## 0.1.0\n\n- First release.\n"; got != want {
		t.Errorf("entry =\n%s", got)
	}
	if again := AddChangelogEntry(got, "0.2.0", []string{"x"}); again != got {
		t.Error("a version already listed was added again")
	}
}

func TestValidateNamesWhatThunderstoreWouldRefuse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	m, _ := w.Create("manifest.json")
	_, _ = m.Write([]byte(`{"name":"Shop Drop","version_number":"1.0","description":"d","dependencies":["BepInEx-BepInExPack-5.4.2305","bad"]}`))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	got := strings.Join(Validate(path), "\n")
	for _, want := range []string{"README.md is missing", "icon.png is missing", "letters, digits and underscores", "three numbers", "website_url", `"bad"`} {
		if !strings.Contains(got, want) {
			t.Errorf("issues lack %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "BepInExPack") {
		t.Errorf("a good dependency was flagged:\n%s", got)
	}
}
