package pack

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

func TestExportCodeRoundTripsThroughAFakeServer(t *testing.T) {
	var posted string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		posted = string(b)
		if r.Method != http.MethodPost || r.URL.Path != "/api/experimental/legacyprofile/create/" ||
			r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("request %s %s %s", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
		}
		_, _ = w.Write([]byte(`{"key":"0123abcd-0123-0123-0123-0123456789ab"}`))
	}))
	t.Cleanup(srv.Close)
	d := Draft{
		Name: "Friends",
		Packages: []Ref{
			{Source: thunderstore, Native: "Alice-MoreCompany", Version: "1.2.3", Disabled: true},
		},
		Configs: []File{{Path: "config/a.cfg", Data: []byte("x=1")}},
	}
	key, err := Code{URL: srv.URL}.ExportCode(t.Context(), d)
	if err != nil || key != "0123abcd-0123-0123-0123-0123456789ab" {
		t.Fatalf("%q %v", key, err)
	}
	got, err := Code{}.Parse(t.Context(), Input{Text: posted})
	if err != nil || got.Name != "Friends" || len(got.Packages) != 1 || got.Packages[0] != d.Packages[0] ||
		len(got.Configs) != 1 {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	d.Packages = append(d.Packages, Ref{Source: "nexus", Native: "1", Version: "1.0.0"})
	if _, err := (Code{URL: srv.URL}).ExportCode(t.Context(), d); err == nil {
		t.Fatal("a Nexus package went into an r2modman code")
	}
}

func TestHostilePacksAreRefused(t *testing.T) {
	bad := func(mod string) string {
		return "profileName: X\nmods:\n  - name: " + mod + "\n    version: {major: 1, minor: 0, patch: 0}\n    enabled: true\n"
	}
	dir := t.TempDir()
	write := func(name string, files map[string]string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, makeZip(t, files), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name   string
		format Format
		in     Input
	}{
		{"mod name that climbs out of its folder", Code{}, Input{Path: write("a.r2z", map[string]string{exportFile: bad("../../etc-passwd")})}},
		{"mod name with a separator", Code{}, Input{Path: write("b.r2z", map[string]string{exportFile: bad(`A-B/../C`)})}},
		{"drive-letter entry", Code{}, Input{Path: write("c.r2z", map[string]string{exportFile: export, `C:/x`: "x"})}},
		{"modpack dependency with a traversal", Modpack{}, Input{Path: write("d.zip", map[string]string{"manifest.json": `{"name":"P","dependencies":["../x-y-1.0.0"]}`})}},
		{"modpack dependency with a bad version", Modpack{}, Input{Path: write("e.zip", map[string]string{"manifest.json": `{"name":"P","dependencies":["A-B-1.0/../x"]}`})}},
	}
	for _, tc := range cases {
		if _, err := tc.format.Parse(t.Context(), tc.in); err == nil {
			t.Errorf("%s: accepted", tc.name)
		}
	}
}

func TestAnUnknownProfileCodeIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	_, err := Code{URL: srv.URL}.Parse(t.Context(), Input{Text: "0123abcd-0123-0123-0123-0123456789ab"})
	if usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("err = %v", err)
	}
}
