package pack

import (
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"testing"

	_ "modernc.org/sqlite"
)

func TestGaleProfileDatabase(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "lethal-company", "profiles", "Crew")
	cfg := filepath.Join(profile, "BepInEx", "config")
	if err := os.MkdirAll(cfg, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "a.cfg"), []byte("[General]"), 0o600); err != nil {
		t.Fatal(err)
	}
	dbFile := filepath.Join(dir, "data.sqlite3")
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE profiles (id INTEGER PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL, game_slug TEXT NOT NULL, mods JSON NOT NULL)`,
		`INSERT INTO profiles (name, path, game_slug, mods) VALUES ('Crew', '` + profile + `', 'lethal-company', '[
			{"enabled":true,"fullName":"BepInEx-BepInExPack-5.4.2100","packageUuid":"u","versionUuid":"v"},
			{"enabled":false,"fullName":"Alice-MoreCompany-1.2.3"},
			{"enabled":true,"name":"Mine","uuid":"x"}]')`,
		`INSERT INTO profiles (name, path, game_slug, mods) VALUES ('Other', '/elsewhere', 'risk-of-rain-2', '[]')`,
	} {
		if _, err := db.ExecContext(t.Context(), q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	g := Gale{DB: dbFile, GameBySlug: func(s string) (string, bool) { return "lc", s == "lethal-company" }}
	if g.Detect(Input{Path: t.TempDir()}) || !g.Detect(Input{Path: profile}) {
		t.Fatal("detection")
	}
	d, err := g.Parse(t.Context(), Input{Path: profile})
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
	if all, err := GaleProfiles(t.Context(), dbFile); err != nil || len(all) != 2 {
		t.Fatalf("profiles = %v, %v", all, err)
	}
}
