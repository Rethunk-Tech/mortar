package pack

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const export = `profileName: Friends
community: lethal-company
mods:
  - name: BepInEx-BepInExPack
    version: {major: 5, minor: 4, patch: 2100}
    enabled: true
  - name: Alice-MoreCompany
    version: {major: 1, minor: 2, patch: 3}
    enabled: false
`

func TestFormats(t *testing.T) {
	r2z := makeZip(t, map[string]string{exportFile: export, "config/a.cfg": "x=1", "BepInEx/plugins/note.txt": "hi"})
	code := codePrefix + "\n" + base64.StdEncoding.EncodeToString(r2z)
	key := "0123abcd-0123-0123-0123-0123456789ab"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/experimental/legacyprofile/get/"+key+"/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(code))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	prof := filepath.Join(dir, "LethalCompany", "profiles", "Mine")
	if err := os.MkdirAll(filepath.Join(prof, "BepInEx", "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	for p, body := range map[string]string{
		filepath.Join(prof, "mods.yml"):                   "- name: Alice-MoreCompany\n  versionNumber: {major: 1, minor: 2, patch: 3}\n  enabled: false\n",
		filepath.Join(prof, "BepInEx", "config", "a.cfg"): "x=1",
	} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pack := filepath.Join(dir, "pack.zip")
	packZip := makeZip(t, map[string]string{
		"manifest.json": `{"name":"Pack","version_number":"1.0.0","dependencies":["Alice-MoreCompany-1.2.3","BepInEx-BepInExPack-5.4.2100"]}`,
		"config/a.cfg":  "x=1",
	})
	if err := os.WriteFile(pack, packZip, 0o600); err != nil {
		t.Fatal(err)
	}
	evil := filepath.Join(dir, "evil.r2z")
	if err := os.WriteFile(evil, makeZip(t, map[string]string{exportFile: export, "../x": "x"}), 0o600); err != nil {
		t.Fatal(err)
	}
	mod := Ref{Source: thunderstore, Native: "Alice-MoreCompany", Version: "1.2.3"}

	tests := []struct {
		name    string
		format  Format
		in      Input
		want    Draft
		wantErr bool
	}{
		{"code text", Code{}, Input{Text: code}, Draft{Name: "Friends", Game: "lethal-company"}, false},
		{"code by key", Code{URL: srv.URL}, Input{Text: key}, Draft{Name: "Friends", Game: "lethal-company"}, false},
		{"r2z file", Code{}, Input{Path: filepath.Join(dir, "x.r2z")}, Draft{}, true},
		{"zip path escape", Code{}, Input{Path: evil}, Draft{}, true},
		{"profile folder", Profile{GameByFolder: func(f string) (string, bool) { return "lethal-company", f == "LethalCompany" }}, Input{Path: prof}, Draft{Name: "Mine", Game: "lethal-company", Packages: []Ref{{Source: thunderstore, Native: "Alice-MoreCompany", Version: "1.2.3", Disabled: true}}}, false},
		{"modpack", Modpack{}, Input{Path: pack}, Draft{Name: "Pack", Packages: []Ref{mod, {Source: thunderstore, Native: "BepInEx-BepInExPack", Version: "5.4.2100"}}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.wantErr && !tc.format.Detect(tc.in) {
				t.Fatal("not detected")
			}
			got, err := tc.format.Parse(t.Context(), tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err %v", err)
			}
			if tc.wantErr {
				return
			}
			if got.Name != tc.want.Name || got.Game != tc.want.Game {
				t.Fatalf("got %+v", got)
			}
			if tc.want.Packages != nil && !slices.Equal(got.Packages, tc.want.Packages) {
				t.Fatalf("packages %+v", got.Packages)
			}
			if len(got.Configs) != 1 || got.Configs[0].Path != "config/a.cfg" || string(got.Configs[0].Data) != "x=1" {
				t.Fatalf("configs %+v", got.Configs)
			}
		})
	}

	d, _ := Code{}.Parse(t.Context(), Input{Text: code})
	if len(d.Packages) != 2 || !d.Packages[1].Disabled || d.Packages[0].Version != "5.4.2100" || len(d.Loose) != 1 {
		t.Fatalf("r2 mods %+v", d)
	}
	if _, err := Read(t.Context(), Input{Text: "hello"}, Code{}, Profile{}, Modpack{}); !errors.Is(err, ErrUnknown) {
		t.Fatalf("unknown input: %v", err)
	}
}
