package datasvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMeasureBudgetsSplitsStoreSharedDeployedAndSaves(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	a, b := "0123456789abcdef", "fedcba9876543210"
	write("profiles/stardew/"+a+"/profile.json", `{"entries":[{"key":"one"},{"key":"two","extraStoreKeys":["extra"]}]}`)
	write("profiles/stardew/"+b+"/profile.json", `{"entries":[{"key":"two"}]}`)
	write("profiles/stardew/"+a+"/mods/x.bin", "12345")
	write("profiles/stardew/"+a+"/saves/s.sav", "123")
	sizes := map[string]int64{"stardew/one": 100, "stardew/two": 40, "stardew/extra": 7}
	got, err := MeasureBudgets(root, sizes, func(_, id string) bool { return id == a }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	first, second := got[0], got[1]
	if first.ID != a || first.Store != 147 || first.Shared != 40 || first.Deployed < 5 || first.Saves < 3 {
		t.Fatalf("first = %+v", first)
	}
	if second.ID != b || second.Store != 40 || second.Shared != 40 || second.Saves != 0 {
		t.Fatalf("second = %+v", second)
	}
}

// A loader kept in the profile (BepInEx) counts under Deployed, byte for byte as du --apparent-size counts its files;
// the profile's own bookkeeping does not.
func TestMeasureBudgetsCountsTheLoaderFilesInTheProfile(t *testing.T) {
	root := t.TempDir()
	id := "0123456789abcdef"
	for rel, n := range map[string]int{
		"profile.json": 2, "history.json": 900, "mods/p/plugin.dll": 10,
		"BepInEx/core/BepInEx.dll": 300, "BepInEx/config/BepInEx.cfg": 40, "winhttp.dll": 25, "doorstop_config.ini": 7,
	} {
		p := filepath.Join(root, "profiles", "lethal-company", id, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		body := strings.Repeat("x", n)
		if rel == "profile.json" {
			body = "{}"
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files := func(string, string) []string {
		return []string{"BepInEx", "winhttp.dll", "doorstop_config.ini", ".doorstop_version"}
	}
	got, err := MeasureBudgets(root, nil, nil, files)
	if err != nil || len(got) != 1 || got[0].Deployed != 10+300+40+25+7 {
		t.Fatalf("budgets %+v, %v", got, err)
	}
}
