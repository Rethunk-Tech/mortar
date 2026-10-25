package devsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

const testTemplate = `{
  "name": "Test template",
  "shortName": ["test-mod", "tm"],
  "sourceName": "ModTemplate",
  "symbols": {
    "description": { "type": "parameter", "datatype": "string", "isRequired": true, "replaces": "ModDescription" },
    "xmlDescription": { "type": "derived", "valueSource": "description", "valueTransform": "xml", "replaces": "ModXmlDescription" },
    "team": { "type": "parameter", "displayName": "Team", "datatype": "string", "isRequired": true, "replaces": "ModTeam" },
    "categories": { "type": "parameter", "datatype": "choice", "allowMultipleValues": true, "defaultValue": "mods",
      "choices": [ { "choice": "mods", "displayName": "Mods" }, { "choice": "items" }, { "choice": "misc" } ] },
    "loaderPackVersion": { "type": "parameter", "datatype": "string", "defaultValue": "5.4.2305", "replaces": "5.4.2305" },
    "year": { "type": "generated", "generator": "now", "replaces": "ModYear" }
  }
}`

const testHost = `{ "symbolInfo": [
  { "id": "description", "maxLength": 20 },
  { "id": "team", "remember": true, "pattern": "^[A-Za-z0-9_]+$" },
  { "id": "loaderPackVersion", "fill": "loaderPackVersion" }
] }`

func templateZip(t *testing.T, template string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"tpl-v1/.template.config/template.json":    template,
		"tpl-v1/.template.config/mortar.host.json": testHost,
		"tpl-v1/README.md":                         "# ModTemplate",
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

// dotnetNew stands in for the tools: `dotnet new <template>` writes an icon into its -o folder and every call is kept.
type dotnetNew struct{ calls [][]string }

func (d *dotnetNew) run(_ context.Context, _, name string, args ...string) ([]byte, error) {
	d.calls = append(d.calls, append([]string{name}, args...))
	if name != "dotnet" || len(args) < 2 || args[0] != "new" {
		return nil, errors.New("no such tool here")
	}
	if args[1] == "install" {
		return nil, nil
	}
	out := args[slices.Index(args, "-o")+1]
	if err := os.MkdirAll(out, 0o750); err != nil {
		return nil, err
	}
	return nil, os.WriteFile(filepath.Join(out, "icon.png"), []byte("icon"), 0o600)
}

func testService(t *testing.T, template string) (*Service, *dotnetNew) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	body := templateZip(t, template)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	t.Cleanup(srv.Close)
	_, profiles := testenv.Stores(t)
	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := &dotnetNew{}
	return &Service{
		Home:         t.TempDir(),
		Settings:     settings.NewService(store),
		Profiles:     profile.NewService(profiles, t.TempDir(), store),
		CacheDir:     t.TempDir(),
		ArchiveURL:   func(components.DevTemplate) string { return srv.URL },
		Run:          d.run,
		LatestLoader: func(context.Context, string) string { return "5.4.2401" },
	}, d
}

func TestTemplateFormAsksForTheTemplatesParametersInOrder(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	form, err := svc.TemplateForm(t.Context(), "lethal-company")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, p := range form.Parameters {
		ids = append(ids, p.ID)
	}
	if !slices.Equal(ids, []string{"description", "team", "categories"}) || form.shortName != "test-mod" {
		t.Fatalf("form = %+v", form)
	}
	team, categories := form.Parameters[1], form.Parameters[2]
	if team.Label != "Team" || form.Parameters[0].Label != "description" {
		t.Errorf("labels %q, %q", team.Label, form.Parameters[0].Label)
	}
	if !team.Remember || team.Pattern == "" || !categories.Multiple || categories.Choices[0] != (Choice{Value: "mods", Label: "Mods"}) || categories.Choices[1].Label != "items" {
		t.Errorf("team %+v, categories %+v", team, categories)
	}
	if form.Parameters[0].MaxLength != 20 || !form.Parameters[0].Required {
		t.Errorf("description %+v", form.Parameters[0])
	}
}

func TestReadFormRefusesPlaceholdersThatOverlap(t *testing.T) {
	// Each would let one value land inside another's text: it holds ModTeam, equals it ignoring case, or sits inside
	// both ModTeam and ModTemplate.
	for _, overlap := range []string{"ModTeamX", "modteam", "ModTe"} {
		svc, _ := testService(t, strings.Replace(testTemplate, `"replaces": "ModDescription"`, `"replaces": "`+overlap+`"`, 1))
		if _, err := svc.TemplateForm(t.Context(), "lethal-company"); err == nil {
			t.Errorf("%s was accepted beside ModTemplate and ModTeam", overlap)
		}
	}
}

func TestCreateRunsDotnetNewAndWiresTheProfile(t *testing.T) {
	svc, d := testService(t, testTemplate)
	parent := t.TempDir()
	p, err := svc.Create(t.Context(), NewMod{
		Game: "lethal-company", Name: "ShopDrop", Parent: parent, Repo: "shop-drop",
		Values: map[string]string{"description": "Lands somewhere.", "team": "Ab", "categories": "items|misc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(parent, "shop-drop")
	var made []string
	for _, c := range d.calls {
		if len(c) > 2 && c[1] == "new" && c[2] == "test-mod" {
			made = c
		}
	}
	for _, want := range [][]string{{"-n", "ShopDrop"}, {"-o", dir}, {"--team", "Ab"}, {"--categories", "items"}, {"--categories", "misc"}, {"--loaderPackVersion", "5.4.2401"}} {
		at := slices.Index(made, want[0])
		if at < 0 || !slices.Contains(made[at:], want[1]) {
			t.Errorf("dotnet new %v lacks %v", made, want)
		}
	}
	body, err := fsx.ReadFile(filepath.Join(dir, ".mortar", "template.json"))
	if err != nil || !strings.Contains(string(body), `"team": "Ab"`) || !strings.Contains(string(body), `"ref": "v0.1.0"`) {
		t.Errorf("record = %s (%v)", body, err)
	}
	gp := svc.Settings.Get().GamePrefs("lethal-company")
	if len(gp.DevProjects) != 1 || gp.DevProjects[0] != p || p.Profile == "" || gp.ExtraModsProfile(p.Output) != p.Profile || p.TemplateIcon == "" {
		t.Fatalf("project %+v, settings %+v", p, gp.DevProjects)
	}
	if _, err := svc.Create(t.Context(), NewMod{Game: "lethal-company", Name: "ShopDrop", Parent: parent, Repo: "shop-drop", Values: map[string]string{"description": "d", "team": "Ab"}}); err == nil {
		t.Error("a second project in the same folder was made")
	}
	if err := svc.Forget("lethal-company", dir); err != nil || len(svc.Projects("lethal-company")) != 0 {
		t.Errorf("forget left %+v (%v)", svc.Projects("lethal-company"), err)
	}
}

func TestCreateChecksValuesBeforeAnythingIsMade(t *testing.T) {
	svc, d := testService(t, testTemplate)
	ok := map[string]string{"description": "d", "team": "Ab"}
	for name, values := range map[string]map[string]string{
		"missing":       {"team": "Ab"},
		"pattern":       {"description": "d", "team": "a b"},
		"too long":      {"description": strings.Repeat("x", 21), "team": "Ab"},
		"dash":          {"description": "--help", "team": "Ab"},
		"not a choice":  {"description": "d", "team": "Ab", "categories": "items|nope"},
		"spans lines":   {"description": "a\nb", "team": "Ab"},
		"accepted base": ok,
	} {
		parent := t.TempDir()
		_, err := svc.Create(t.Context(), NewMod{Game: "lethal-company", Name: "ShopDrop", Parent: parent, Repo: "r", Values: values})
		if (err == nil) != (name == "accepted base") {
			t.Errorf("%s: %v", name, err)
		}
		if name != "accepted base" {
			if _, statErr := os.Stat(filepath.Join(parent, "r")); statErr == nil {
				t.Errorf("%s: a project was made", name)
			}
		}
	}
	_ = d
}

func TestNewModRefusesUnsafeNames(t *testing.T) {
	ok := NewMod{Name: "ShopDrop", Parent: "/p", Repo: "r"}
	if err := ok.check(); err != nil {
		t.Fatal(err)
	}
	for _, m := range []func(*NewMod){
		func(m *NewMod) { m.Name = "shop drop" },
		func(m *NewMod) { m.Repo = ".." },
		func(m *NewMod) { m.Repo = "a/b" },
		func(m *NewMod) { m.Repo = "-c" },
		func(m *NewMod) { m.Parent = "rel" },
		func(m *NewMod) { m.GitHub, m.Owner = true, "--help" },
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
	f, err := fsx.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	icon, _ := w.Create("icon.png")
	_, _ = icon.Write([]byte("not a png"))
	m, _ := w.Create("manifest.json")
	_, _ = m.Write([]byte(`{"name":"Shop Drop","version_number":"1.0","description":"d","dependencies":["BepInEx-BepInExPack-5.4.2305","bad"]}`))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	got := strings.Join(Validate(path), "\n")
	for _, want := range []string{"README.md is missing", "letters, digits and underscores", "three numbers", "website_url", `"bad"`} {
		if !strings.Contains(got, want) {
			t.Errorf("issues lack %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "256×256") {
		t.Errorf("an icon that is not a PNG passed:\n%s", got)
	}
	if h := zipFileHash(path, "icon.png"); h != fmt.Sprintf("%x", sha256.Sum256([]byte("not a png"))) || zipFileHash(path, "none") != "" {
		t.Errorf("icon hash = %q", h)
	}
	if strings.Contains(got, "BepInExPack") {
		t.Errorf("a good dependency was flagged:\n%s", got)
	}
}
