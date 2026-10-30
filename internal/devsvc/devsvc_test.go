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
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
    "netcode": { "type": "parameter", "datatype": "bool", "defaultValue": "false" },
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
	if !slices.Equal(ids, []string{"description", "team", "categories", "netcode"}) || form.Parameters[3].Kind != "bool" || form.shortName != "test-mod" {
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
		Values: map[string]string{"description": "Lands somewhere.", "team": "Ab", "categories": "items|misc", "netcode": "true"},
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
	for _, want := range [][]string{{"-n", "ShopDrop"}, {"-o", dir}, {"--team", "Ab"}, {"--categories", "items"}, {"--categories", "misc"}, {"--loaderPackVersion", "5.4.2401"}, {"--netcode", "true"}} {
		at := slices.Index(made, want[0])
		if at < 0 || !slices.Contains(made[at:], want[1]) {
			t.Errorf("dotnet new %v lacks %v", made, want)
		}
	}
	body, err := safefs.ReadFile(filepath.Join(dir, ".mortar", "template.json"))
	if err != nil || !strings.Contains(string(body), `"team": "Ab"`) || !strings.Contains(string(body), `"ref": "v0.1.1"`) {
		t.Errorf("record = %s (%v)", body, err)
	}
	gp := svc.Settings.Get().GamePrefs("lethal-company")
	if len(gp.DevProjects) != 1 || gp.DevProjects[0] != p || p.Profile == "" || gp.ExtraModsProfile(p.Output) != p.Profile || p.TemplateIcon == "" || p.Untrusted {
		t.Fatalf("project %+v, settings %+v", p, gp.DevProjects)
	}
	if _, err := svc.Create(t.Context(), NewMod{Game: "lethal-company", Name: "ShopDrop", Parent: parent, Repo: "shop-drop", Values: map[string]string{"description": "d", "team": "Ab"}}); err == nil {
		t.Error("a second project in the same folder was made")
	}
	trash := t.TempDir()
	t.Setenv("XDG_DATA_HOME", trash)
	if err := svc.Trash("lethal-company", dir); err != nil || len(svc.Projects("lethal-company")) != 0 || safefs.IsDir(dir) {
		t.Errorf("trash left %+v, folder there %v (%v)", svc.Projects("lethal-company"), safefs.IsDir(dir), err)
	}
	if runtime.GOOS != "windows" && (!safefs.IsFile(filepath.Join(trash, "Trash", "files", filepath.Base(dir), ".mortar", "template.json")) || !safefs.IsFile(filepath.Join(trash, "Trash", "info", filepath.Base(dir)+".trashinfo"))) {
		t.Error("the project did not land in the freedesktop trash with its .trashinfo, so the file manager cannot restore it")
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
		"not a bool":    {"description": "d", "team": "Ab", "netcode": "yes"},
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
	ok := NewMod{Name: "ShopDrop", Parent: t.TempDir(), Repo: "r"}
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
	if v := nextVersions(""); !reflect.DeepEqual(v, Versions{Patch: "0.1.0", Minor: "0.1.0", Major: "0.1.0"}) {
		t.Errorf("first = %+v", v)
	}
	if v := nextVersions("1.4.9"); !reflect.DeepEqual(v, Versions{Current: "1.4.9", Patch: "1.4.10", Minor: "1.5.0", Major: "2.0.0"}) {
		t.Errorf("after 1.4.9 = %+v", v)
	}
	if got := changeLines("Fix the drop point\n\nRelease 0.2.0\nRelease notes in README\n"); !slices.Equal(got, []string{"Fix the drop point", "Release notes in README"}) {
		t.Errorf("changes = %q", got)
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
	f, err := safefs.Create(path)
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

// A new development profile copies the named profile's mods, so a profile that is not there stops the start.
func TestCreateCopiesModsIntoTheNewProfile(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	_, err := svc.Create(t.Context(), NewMod{
		Game: "lethal-company", Name: "ShopDrop", Parent: t.TempDir(), Repo: "shop-drop", CopyFrom: "missing",
		Values: map[string]string{"description": "Lands somewhere.", "team": "Ab"},
	})
	if err == nil {
		t.Fatal("copied from a profile that does not exist")
	}
}

// writeAll writes a file and the folders above it.
func writeAll(path string, body []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, body, mode)
}

func projectDir(t *testing.T, name string, record bool) string {
	t.Helper()
	dir := t.TempDir()
	if err := writeAll(filepath.Join(dir, "src", name, name+".csproj"), []byte("<Project/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if record {
		if err := writeAll(filepath.Join(dir, ".mortar", "template.json"), []byte(`{"name": "`+name+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestAddTakesAnExistingTemplateProject(t *testing.T) {
	svc, run := testService(t, testTemplate)
	const game = "lethal-company"
	props := []byte("<Project/>")
	kept := projectDir(t, "Kept", true)
	if err := writeAll(filepath.Join(kept, "Config.Build.user.props"), props, 0o600); err != nil {
		t.Fatal(err)
	}
	recorded := projectDir(t, "Twice", false)
	added, err := svc.Add(t.Context(), game, recorded, "")
	if err != nil {
		t.Fatal(err)
	}
	built := func() bool {
		return slices.ContainsFunc(run.calls, func(c []string) bool { return slices.Equal(c[:2], []string{"dotnet", "build"}) })
	}
	if !added.Untrusted || built() {
		t.Errorf("an added project is built before the user agreed: %+v %v", added, run.calls)
	}
	for name, do := range map[string]func() error{
		"build": func() error { _, err := svc.Build(t.Context(), game, recorded); return err },
		"pack":  func() error { _, err := svc.Pack(t.Context(), game, recorded); return err },
	} {
		if err := do(); usererr.KindOf(err) != usererr.Unconfirmed {
			t.Errorf("%s of an untrusted project: %v", name, err)
		}
	}
	if built() {
		t.Fatalf("a refused build ran dotnet: %v", run.calls)
	}
	if safefs.UnderRoot(recorded, added.Output) || slices.ContainsFunc(svc.Settings.Get().GamePrefs(game).ExtraModsFolderList(), func(d string) bool { return safefs.UnderRoot(recorded, d) }) {
		t.Errorf("an untrusted project's committed output is an installed mods folder: %q", added.Output)
	}
	if err := svc.Trust(game, recorded); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Build(t.Context(), game, recorded); err != nil || !built() {
		t.Errorf("a trusted project does not build: %v %v", err, run.calls)
	}
	if got := svc.Projects(game)[0]; got.Untrusted || got.Output != filepath.Join(recorded, "artifacts", "bin", got.Name, "debug") {
		t.Errorf("trust was not kept or did not move the output: %+v", got)
	}
	for _, c := range []struct {
		name, dir string
		ok        bool
	}{
		{"valid, name from the folder", projectDir(t, "ShopDrop", false), true},
		{"valid, name from the record", projectDir(t, "Named", true), true},
		{"existing user props", kept, true},
		{"not a project", t.TempDir(), false},
		{"already recorded", recorded, false},
		{"relative", "mods/here", false},
	} {
		p, err := svc.Add(t.Context(), game, c.dir, "")
		if (err == nil) != c.ok {
			t.Errorf("%s: err = %v", c.name, err)
			continue
		}
		if c.ok && (p.Dir != c.dir || p.Profile == "" || safefs.UnderRoot(c.dir, p.Output)) {
			t.Errorf("%s: project %+v", c.name, p)
		}
	}
	if all, err := svc.Profiles.List(game); err != nil || len(all) == 0 || !all[0].SeparateSaves {
		t.Errorf("a development profile shares the player's saves: %v", err)
	}
	if got, _ := safefs.ReadFile(filepath.Join(kept, "Config.Build.user.props")); !bytes.Equal(got, props) {
		t.Errorf("user props rewritten: %s", got)
	}
	if _, err := svc.Add(t.Context(), game, projectDir(t, "Other", false), "missing"); err == nil {
		t.Error("added into a profile that does not exist")
	}
}

func TestParseRepoReadsOwnerAndName(t *testing.T) {
	for in, want := range map[string]string{
		"Rethunk-Tech/mod":             "Rethunk-Tech/mod",
		"https://github.com/a/b.c.git": "a/b.c",
		"git@github.com:a/b-c.git":     "a/b-c",
		"https://github.com/a/b/":      "a/b",
		"":                             "",
		"just-a-name":                  "",
		"https://gitlab.com/a/b":       "",
		"a/b/c":                        "",
		"-a/b":                         "",
		"a/../b":                       "",
		"a/-b":                         "",
		"a/--upload-pack=x":            "",
	} {
		owner, name, err := ParseRepo(in)
		if got := owner + "/" + name; (want == "") != (err != nil) || (err == nil && got != want) {
			t.Errorf("%q -> %q, %v", in, got, err)
		}
	}
}

func TestCloneRemovesWhatIsNotAProject(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	parent := t.TempDir()
	clone := func(files bool) func(context.Context, string, string, ...string) ([]byte, error) {
		return func(_ context.Context, dir, name string, args ...string) ([]byte, error) {
			if name == "gh" && args[0] == "api" {
				return nil, errors.New("signed out")
			}
			if name != "git" || args[0] != "clone" || args[1] != "--" || args[2] != "https://github.com/o/shop.git" {
				return nil, errors.New("unexpected " + name)
			}
			out := filepath.Join(dir, args[3])
			if files {
				return nil, writeAll(filepath.Join(out, "src", "Shop", "Shop.csproj"), []byte("<Project/>"), 0o644)
			}
			return nil, os.MkdirAll(out, 0o750)
		}
	}
	svc.Run = clone(false)
	if _, err := svc.Clone(t.Context(), "lethal-company", "o/shop", parent, ""); err == nil {
		t.Fatal("a repository without the template's layout was added")
	}
	if _, err := os.Stat(filepath.Join(parent, "shop")); err == nil {
		t.Error("the refused clone was left behind")
	}
	svc.Run = clone(true)
	p, err := svc.Clone(t.Context(), "lethal-company", "https://github.com/o/shop", parent, "")
	if err != nil || p.Name != "Shop" || p.Dir != filepath.Join(parent, "shop") {
		t.Fatalf("clone = %+v, %v", p, err)
	}
	if _, err := svc.Clone(t.Context(), "lethal-company", "o/shop", parent, ""); err == nil {
		t.Error("cloned over an existing folder")
	}
}

// MORTAR_DEV_TEMPLATE points New mod at a template that isn't published yet: a folder, read fresh each time, or a zip.
func TestTemplateEnvTakesALocalFolderOrZip(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.ArchiveURL = nil
	tpl := components.DevTemplate{Repo: "o/n", Ref: "v1"}

	src := t.TempDir()
	for name, body := range map[string]string{
		".template.config/template.json":    testTemplate,
		".template.config/mortar.host.json": testHost,
		"README.md":                         "first",
		".git/HEAD":                         "ref: refs/heads/main",
	} {
		path := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(TemplateEnv, src)
	dir, err := svc.templateDir(t.Context(), tpl)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		t.Fatal("the template's .git came along")
	}
	if err := os.WriteFile(filepath.Join(src, "README.md"), []byte("edited"), 0o600); err != nil {
		t.Fatal(err)
	}
	if dir, err = svc.templateDir(t.Context(), tpl); err != nil {
		t.Fatal(err)
	}
	if b, _ := safefs.ReadFile(filepath.Join(dir, "README.md")); string(b) != "edited" {
		t.Fatalf("a local folder was not read again: %q", b)
	}

	zipPath := filepath.Join(t.TempDir(), "template.zip")
	if err := os.WriteFile(zipPath, templateZip(t, testTemplate), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(TemplateEnv, zipPath)
	if dir, err = svc.templateDir(t.Context(), tpl); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(templateFile))); err != nil {
		t.Fatalf("the zip's one folder was not unwrapped: %v", err)
	}

	t.Setenv(TemplateEnv, filepath.Join(t.TempDir(), "missing"))
	if _, err := svc.templateDir(t.Context(), tpl); err == nil {
		t.Fatal("a missing template was accepted")
	}
}

// A fake gh: the signed-in user with one organisation, and a GitHub holding only o/taken.
func ghAPI(_ context.Context, _, name string, args ...string) ([]byte, error) {
	if name != "gh" || args[0] != "api" {
		return nil, errors.New("no such tool here")
	}
	switch strings.Join(args[1:], " ") {
	case `user --jq ` + ownerFields:
		return []byte("me\thttps://a/me\n"), nil
	case `--paginate user/orgs --jq .[]|` + ownerFields:
		return []byte("o\thttps://a/o\n"), nil
	case `users/o --jq ` + ownerFields:
		return []byte("o\thttps://a/o\n"), nil
	case "repos/o/taken --jq .id":
		return []byte("1\n"), nil
	}
	return nil, errors.New("gh: Not Found (HTTP 404)")
}

func TestGitHubOwnersAndRepoCheck(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = ghAPI
	tools := svc.Tools(t.Context(), "lethal-company")
	if tools.GitHubUser != "me" || len(tools.GitHubOwners) != 2 || tools.GitHubOwners[1] != (GitHubOwner{Login: "o", Avatar: "https://a/o"}) {
		t.Fatalf("tools = %+v", tools)
	}
	for repo, taken := range map[string]bool{"taken": true, "free": false} {
		check, err := svc.CheckRepo(t.Context(), "o", repo)
		if err != nil || check.Owner == nil || check.Taken != taken {
			t.Errorf("%s: %+v, %v", repo, check, err)
		}
	}
	if check, err := svc.CheckRepo(t.Context(), "nobody", "free"); err != nil || check.Owner != nil {
		t.Errorf("missing owner: %+v, %v", check, err)
	}
	if _, err := svc.CheckRepo(t.Context(), "--help", "x"); err == nil {
		t.Error("an owner gh would read as a flag was looked up")
	}
}

// A release with a GitHub remote reaches it, so the template's CI publishes; without one it stays a local tag.
func TestReleasePushesOnlyToGitHub(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = nil
	git := func(dir string, args ...string) {
		t.Helper()
		if _, err := svc.run(t.Context(), dir, "git", args...); err != nil {
			t.Fatal(err)
		}
	}
	project := func(name string) string {
		dir := filepath.Join(t.TempDir(), name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		git(dir, "init", "-q")
		git(dir, "commit", "-q", "--allow-empty", "-m", "start")
		if err := svc.Settings.AddDevProject("lethal-company", settings.DevProject{Dir: dir, Name: name}); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")

	local := project("Local")
	if remote, err := svc.Release(t.Context(), "lethal-company", local, "0.1.0", []string{"First"}); err != nil || remote != "" {
		t.Fatalf("local release = %q, %v", remote, err)
	}

	hub := filepath.Join(t.TempDir(), "github.com", "o", "shop.git")
	if err := os.MkdirAll(hub, 0o750); err != nil {
		t.Fatal(err)
	}
	git(hub, "init", "-q", "--bare")
	shared := project("Shared")
	git(shared, "remote", "add", "origin", hub)
	if v, err := svc.NextVersions(t.Context(), "lethal-company", shared); err != nil || v.GitHub != "origin" {
		t.Fatalf("versions = %+v, %v", v, err)
	}
	if remote, err := svc.Release(t.Context(), "lethal-company", shared, "0.1.0", []string{"First"}); err != nil || remote != "origin" {
		t.Fatalf("github release = %q, %v", remote, err)
	}
	git(hub, "rev-parse", "-q", "--verify", "refs/tags/v0.1.0")
	if remote, err := svc.Release(t.Context(), "lethal-company", shared, "0.1.0", []string{"First"}); err != nil || remote != "origin" {
		t.Fatalf("retried release = %q, %v", remote, err)
	}
}

func TestAnOversizedTemplateZipIsRefused(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	chunk := make([]byte, 1<<20)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for range maxTemplateZip/len(chunk) + 1 {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	svc.ArchiveURL = func(components.DevTemplate) string { return srv.URL }
	_, err := svc.TemplateForm(t.Context(), "lethal-company")
	if err == nil || !strings.Contains(err.Error(), "downloading the template") {
		t.Fatalf("a template zip past maxTemplateZip was accepted: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(svc.CacheDir, "dev-templates")); len(entries) != 0 {
		t.Fatalf("the refused template was cached: %v", entries)
	}
}

func TestCreateLeavesNoFolderBehindWhenDotnetNewFails(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	svc.Run = func(_ context.Context, _, name string, args ...string) ([]byte, error) {
		if len(args) > 1 && args[0] == "new" && args[1] != "install" {
			if err := os.MkdirAll(args[slices.Index(args, "-o")+1], 0o750); err != nil {
				return nil, err
			}
			return nil, errors.New("template failed")
		}
		return nil, nil
	}
	parent := t.TempDir()
	m := NewMod{
		Game: "lethal-company", Name: "ShopDrop", Parent: parent, Repo: "shop-drop",
		Values: map[string]string{"description": "Lands somewhere.", "team": "Ab", "categories": "items|misc", "netcode": "true"},
	}
	if _, err := svc.Create(t.Context(), m); err == nil {
		t.Fatal("create succeeded")
	}
	if _, err := os.Lstat(filepath.Join(parent, "shop-drop")); err == nil {
		t.Error("the failed project's folder was left behind")
	}
}

func TestAddReplacesACommittedGameRootDir(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := projectDir(t, "Evil", true)
	props := filepath.Join(dir, "Config.Build.user.props")
	if err := writeAll(props, []byte("<Project><PropertyGroup><GameRootDir>/evil/</GameRootDir><Other>x</Other></PropertyGroup></Project>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Add(t.Context(), "lethal-company", dir, ""); err != nil {
		t.Fatal(err)
	}
	body, _ := safefs.ReadFile(props)
	if strings.Contains(string(body), "evil") || !strings.Contains(string(body), "<Other>x</Other>") {
		t.Errorf("props = %s", body)
	}
}

func TestAFailedEnrollLeavesNoDevelopmentProfile(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	const game = "lethal-company"
	dir := projectDir(t, "Twice", false)
	// A cache path that is a file makes the untrusted output folder impossible, after the profile was made.
	svc.CacheDir = filepath.Join(dir, "src", "Twice", "Twice.csproj")
	if _, err := svc.Add(t.Context(), game, dir, ""); err == nil {
		t.Fatal("add succeeded")
	}
	if all, err := svc.Profiles.List(game); err != nil || len(all) != 0 {
		t.Errorf("profiles left behind: %v %v", all, err)
	}
}
