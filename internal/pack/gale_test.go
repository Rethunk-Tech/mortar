package pack

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGaleProfileFolder(t *testing.T) {
	data := t.TempDir()
	dir := filepath.Join(data, "lethal-company", "profiles", "Crew")
	for path, body := range map[string]string{
		"profile.json": `{"mods":[
			{"fullName":"BepInEx-BepInExPack-5.4.2100","enabled":true,"packageUuid":"u","versionUuid":"v"},
			{"fullName":"Alice-MoreCompany-1.2.3","enabled":false},
			{"name":"Mine","uuid":"x","enabled":true}]}`,
		"BepInEx/config/a.cfg": "[General]",
	} {
		p := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	g := Gale{GameBySlug: func(s string) (string, bool) { return "lc", s == "lethal-company" }}
	if g.Detect(Input{Path: t.TempDir()}) || !g.Detect(Input{Path: dir}) {
		t.Fatal("detection")
	}
	d, err := g.Parse(t.Context(), Input{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	want := []Ref{
		{Source: thunderstore, Native: "BepInEx-BepInExPack", Version: "5.4.2100"},
		{Source: thunderstore, Native: "Alice-MoreCompany", Version: "1.2.3", Disabled: true},
	}
	if d.Name != "Crew" || d.Game != "lc" || !slices.Equal(d.Packages, want) || len(d.Configs) != 1 || d.Configs[0].Path != "config/a.cfg" {
		t.Fatalf("draft = %+v", d)
	}
}
