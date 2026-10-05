package pack

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func fuzzZip(files map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for n, c := range files {
		w, _ := zw.Create(n)
		_, _ = w.Write([]byte(c))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func FuzzParseR2Zip(f *testing.F) {
	f.Add(fuzzZip(map[string]string{exportFile: "profileName: p\ncommunity: c\nmods:\n- name: A-B-1.0.0\n  enabled: true\n", "config/a.cfg": "x"}))
	f.Add(fuzzZip(map[string]string{exportFile: ":\n- ["}))
	f.Add(fuzzZip(map[string]string{"../evil": "x"}))
	f.Add([]byte("PK"))
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := parseR2Zip(data)
		if err != nil {
			return
		}
		for _, fl := range append(append([]File{}, d.Configs...), d.Loose...) {
			if strings.HasPrefix(fl.Path, "/") || fl.Path == ".." || strings.HasPrefix(fl.Path, "../") {
				t.Fatalf("path escapes: %q", fl.Path)
			}
		}
	})
}

func FuzzModpackManifest(f *testing.F) {
	f.Add(fuzzZip(map[string]string{"manifest.json": `{"name":"n","version_number":"1.0.0","dependencies":["A-B-1.0.0"]}`}))
	f.Add(fuzzZip(map[string]string{"manifest.json": `{`}))
	f.Fuzz(func(t *testing.T, data []byte) {
		files, err := readZip(data)
		if err != nil {
			return
		}
		for name := range files {
			if strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") {
				t.Fatalf("path escapes: %q", name)
			}
		}
	})
}

// FuzzGaleRows holds that whatever Gale's database rows contain, listing and parsing a profile never panic and
// every package that comes out is a well-formed Thunderstore name.
func FuzzGaleRows(f *testing.F) {
	f.Add("Crew", "lethal-company", `[{"enabled":true,"fullName":"A-B-1.0.0"},{"name":"Mine"}]`)
	f.Add("", "", `[`)
	f.Add("x", "y", `[{"fullName":"-"},{"fullName":"a-b"},{"fullName":"a-b-c-d","enabled":null}]`)
	dir := f.TempDir()
	profile := filepath.Join(dir, "p")
	if err := os.MkdirAll(profile, 0o750); err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, name, slug, mods string) {
		dbFile := filepath.Join(t.TempDir(), "data.sqlite3")
		db, err := sql.Open("sqlite", dbFile)
		if err != nil {
			t.Skip()
		}
		defer func() { _ = db.Close() }()
		if _, err := db.ExecContext(t.Context(), `CREATE TABLE profiles (id INTEGER PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL, game_slug TEXT NOT NULL, mods JSON NOT NULL)`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(t.Context(), `INSERT INTO profiles (name, path, game_slug, mods) VALUES (?, ?, ?, ?)`, name, profile, slug, mods); err != nil {
			t.Skip()
		}
		g := Gale{DB: dbFile}
		if _, err := GaleProfiles(t.Context(), dbFile); err != nil {
			t.Fatal(err)
		}
		d, err := g.Parse(t.Context(), Input{Path: profile})
		if err != nil {
			return
		}
		for _, p := range d.Packages {
			if !nativeID.MatchString(p.Native) {
				t.Fatalf("malformed package %q", p.Native)
			}
		}
	})
}
