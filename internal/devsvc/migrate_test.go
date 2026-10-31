package devsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestMigrateRecordsWritesTheStandardShapeOnce(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	const oldTemplate = `{"template":"T/t","ref":"main","name":"Mod","values":{"a":"b"},"keep":{"x":1}}`
	const newTemplate = "{\n  \"template\": \"T/t\",\n  \"ref\": \"main\",\n  \"name\": \"Mod\",\n  \"values\": {\n    \"a\": \"b\"\n  },\n  \"game\": \"lethal-company\",\n  \"templateIcon\": \"abc\"\n}\n"
	cases := []struct {
		name, file, before, icon, want string
	}{
		{"template gains game and icon", "template.json", `{"template":"T/t","ref":"main","name":"Mod","values":{"a":"b"}}`, "abc", newTemplate},
		{
			"unknown template key survives", "template.json", oldTemplate, "abc",
			"{\n  \"game\": \"lethal-company\",\n  \"keep\": {\n    \"x\": 1\n  },\n  \"name\": \"Mod\",\n  \"ref\": \"main\",\n  \"template\": \"T/t\",\n  \"templateIcon\": \"abc\",\n  \"values\": {\n    \"a\": \"b\"\n  }\n}\n",
		},
		{
			"0.3.8 pack record", "project.json", `{"kind":"posters","name":"Mod","author":"me","team":"Mortar"}`, "abc",
			"{\n  \"team\": \"Mortar\",\n  \"game\": \"lethal-company\"\n}\n",
		},
		{"new template untouched", "template.json", newTemplate, "", newTemplate},
		{"new record untouched", "project.json", "{\n  \"team\": \"Mortar\",\n  \"game\": \"lethal-company\"\n}\n", "", "{\n  \"team\": \"Mortar\",\n  \"game\": \"lethal-company\"\n}\n"},
		{"damaged file left", "project.json", `{"team":`, "", `{"team":`},
		// Last: the one project that keeps its icon in settings, since its template.json cannot be read.
		{"damaged template keeps the kept icon", "template.json", `{"template":`, "abc", `{"template":`},
	}
	var dirs []string
	for _, c := range cases {
		dir := adoptFolder(t, "Mod", map[string]string{".mortar/" + c.file: c.before})
		dirs = append(dirs, dir)
		if err := svc.Settings.AddDevProject("lethal-company", settings.DevProject{Dir: dir, Name: "Mod", TemplateIcon: c.icon, Output: dir}); err != nil {
			t.Fatal(err)
		}
	}
	gone := filepath.Join(t.TempDir(), "gone")
	if err := svc.Settings.AddDevProject("lethal-company", settings.DevProject{Dir: gone, Name: "Gone", Output: gone}); err != nil {
		t.Fatal(err)
	}
	svc.MigrateRecords()
	for i, c := range cases {
		path := filepath.Join(dirs[i], ".mortar", c.file)
		if got, _ := safefs.ReadFile(path); string(got) != c.want {
			t.Errorf("%s: got %q", c.name, got)
		}
	}
	for _, p := range svc.Projects("lethal-company") {
		if p.TemplateIcon != "" && p.Dir != dirs[len(dirs)-1] {
			t.Errorf("%s still keeps the icon in settings", p.Dir)
		}
	}
	stamp := func(i int) int64 {
		st, err := os.Stat(filepath.Join(dirs[i], ".mortar", cases[i].file))
		if err != nil {
			t.Fatal(err)
		}
		return st.ModTime().UnixNano()
	}
	before := stamp(0)
	svc.MigrateRecords()
	if stamp(0) != before {
		t.Error("a second run rewrote a migrated file")
	}
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Errorf("a missing folder was created: %v", err)
	}
}
