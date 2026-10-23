package profile

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// An archive the plain installer gets for a game that is not modded through SMAPI is refused in words that do not
// name SMAPI, and never opens a SMAPI folder ask.
func TestNonSMAPIGamesRefuseAnUnrecognisedArchiveWithoutSMAPIWords(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, game string
		files      map[string]string
	}{
		{"hades-ii lua folder", "hades-ii", map[string]string{"Foo/main.lua": "x", "Foo/modfile.txt": "x"}},
		{"balatro lua folder", "balatro", map[string]string{"Foo/main.lua": "x", "Foo/modfile.txt": "x"}},
		{"lethal-company config only", "lethal-company", map[string]string{"cfg.cfg": "x"}},
		{"lethal-company manifest without a version", "lethal-company", map[string]string{"manifest.json": `{"name":"Foo"}`, "icon.png": "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t)
			p, err := e.Create(tc.game, "P")
			if err != nil {
				t.Fatal(err)
			}
			zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "Foo.zip"), tc.files)
			for name, install := range map[string]func() (InstallResult, error){
				"archive": func() (InstallResult, error) { return e.InstallArchive(t.Context(), tc.game, p.ID, zip) },
				"folder": func() (InstallResult, error) {
					dir := t.TempDir()
					for rel, body := range tc.files {
						writeFile(t, dir, rel, body)
					}
					return e.InstallFolder(t.Context(), tc.game, p.ID, dir)
				},
			} {
				res, err := install()
				var ie *InstallError
				if !errors.As(err, &ie) || strings.Contains(ie.Msg, "SMAPI") || !errors.As(err, new(*NoPackageError)) {
					t.Fatalf("%s: err = %v", name, err)
				}
				if res.Remap != nil || len(res.Profile.Entries) != 0 {
					t.Fatalf("%s: result = %+v", name, res)
				}
				if cur, rerr := e.Get(tc.game, p.ID); rerr != nil || len(cur.Entries) != 0 {
					t.Fatalf("%s: profile = %+v, %v", name, cur, rerr)
				}
			}
		})
	}
}
