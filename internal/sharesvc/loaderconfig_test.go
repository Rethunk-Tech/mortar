package sharesvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/share"

	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

// A Lethal Company profile's BepInEx config travels in its .mortar file and lands in the importing profile; the bridge's
// state file, binaries and files outside the config folder stay behind.
func TestLoaderConfigsRoundTripThroughAMortarFile(t *testing.T) {
	s, _ := newService(t, true)
	src, err := s.d.Profiles.Create("lethal-company", "Lobby")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.d.Profiles.ProfileDir("lethal-company", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"BepInEx/config/Sigurd.CSync.cfg":           "[A]\nx = 1\n",
		"BepInEx/config/Sub/more.json":              `{"y":2}`,
		"BepInEx/config/mortar-bepinex-bridge.json": `{"token":"secret"}`,
		"BepInEx/config/cache.dll":                  "MZ",
		"BepInEx/plugins/stray.cfg":                 "z",
	}
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	payload, _, err := s.ExportBytes("lethal-company", src.ID, share.DefaultInclude())
	if err != nil {
		t.Fatal(err)
	}
	pv, err := s.previewBytes(t.Context(), "lethal-company", payload, "")
	if err != nil || pv.Settings != 2 {
		t.Fatalf("preview settings %d, %v", pv.Settings, err)
	}
	res, err := s.Import(t.Context(), "lethal-company", sessionOf(s), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.d.Profiles.ProfileDir("lethal-company", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]bool{
		"BepInEx/config/Sigurd.CSync.cfg": true, "BepInEx/config/Sub/more.json": true,
		"BepInEx/config/mortar-bepinex-bridge.json": false, "BepInEx/config/cache.dll": false, "BepInEx/plugins/stray.cfg": false,
	} {
		b, err := fsx.ReadFile(filepath.Join(got, filepath.FromSlash(rel)))
		if (err == nil) != want || (want && string(b) != files[rel]) {
			t.Errorf("%s arrived %v (%q), want %v", rel, err == nil, b, want)
		}
	}
}
