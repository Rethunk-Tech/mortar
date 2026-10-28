package profile

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

type profileCreator interface {
	Create(game, name string) (Profile, error)
}

func mustCreate(t testing.TB, s profileCreator, name string) Profile {
	t.Helper()
	p, err := s.Create("stardew", name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// envWithAB is a fresh env whose store holds the SMAPI mods Me.A (local-a) and Me.B (local-b).
func envWithAB(t *testing.T) env {
	t.Helper()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{"manifest.json": manifestJSON("Me.A")})
	e.item(t, "local-b", map[string]string{"manifest.json": manifestJSON("Me.B")})
	return e
}

// withSettings is a Service over e with settings of its own, and those settings.
func (e env) withSettings(t *testing.T) (*Service, *settings.Store) {
	t.Helper()
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return NewService(e.Store, t.TempDir(), st), st
}

// lcZip is a new Lethal Company profile on e and a zip called name holding files, to install into it.
func (e env) lcZip(t *testing.T, name string, files map[string]string) (Profile, string) {
	t.Helper()
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	return p, testfs.WriteZip(t, filepath.Join(t.TempDir(), name), files)
}

// installedFolderPackage is a fresh env whose folder game profile installed a.zip of files as CurseForge file 10.
func installedFolderPackage(t *testing.T, files map[string]string) (env, InstallResult, error) {
	t.Helper()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	res, err := e.InstallSource(t.Context(), folderGame, p.ID, zipOf(t, "a.zip", files), cfSource(10))
	return e, res, err
}

// editProfileJSON rewrites the profile.json at path as edit changes its fields, as a hand edit or an older release
// would have left it.
func editProfileJSON(t *testing.T, path string, edit func(doc map[string]any)) {
	t.Helper()
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	edit(doc)
	if raw, err = json.Marshal(doc); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// addZip adds the store item key to the Stardew profile as a local archive named key.zip.
func (e env) addZip(t *testing.T, profileID, key string) {
	t.Helper()
	if _, err := e.AddEntry("stardew", profileID, key, Source{Kind: KindLocal, Name: key + ".zip"}); err != nil {
		t.Fatal(err)
	}
}

// disable turns the mod id off wherever the Stardew profile holds it.
func (e env) disable(t *testing.T, profileID string, id mod.ID) {
	t.Helper()
	if _, err := e.SetModEnabled("stardew", profileID, "", id, false); err != nil {
		t.Fatal(err)
	}
}
