package sharesvc

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

const codeKey = "019c9378-d2e6-4fec-5a2f-e4b22f8cf1d1"

const packExport = "profileName: Just some mods\nmods:\n" +
	"  - {name: Alice-MoreCompany, version: {major: 1, minor: 2, patch: 3}, enabled: true}\n" +
	"  - {name: Bob-Gone, version: {major: 9, minor: 9, patch: 9}, enabled: false}\n"

func r2z(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"export.r2x": packExport, "config/alice.cfg": "[General]\n", "BepInEx/plugins/run.dll": "MZ",
		"BepInEx/plugins/Alice-MoreCompany/extra.dll": "MZ", "BepInEx/plugins/Mine/Mine.dll": "MZ", "BepInEx/plugins/Mine/mine.json": "{}",
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// packService is a lethal-company share service whose r2modman codes come from a fake Thunderstore and whose index
// cannot resolve Bob-Gone.
func packService(t *testing.T) (*Service, *recorder) {
	t.Helper()
	t.Setenv("MORTAR_ENABLE_GAMES", "lethal-company")
	testfs.DataHome(t)
	body := "#r2modman\n" + base64.StdEncoding.EncodeToString(r2z(t))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/experimental/legacyprofile/get/"+codeKey+"/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	_, profiles := testenv.Stores(t)
	rec := &recorder{}
	r := testResolver(nil, nil)
	return NewService(Deps{
		Profiles: profiles, Queue: rec, Meta: r.meta, Files: r.files, PackCode: pack.Code{URL: srv.URL},
		SignedIn: func() bool { return true }, Premium: func() bool { return false },
		Env: func(string) problems.Environment { return problems.Environment{} },
		Unresolvable: func(_ context.Context, _, _ string, reqs []queue.Request) map[string]string {
			out := map[string]string{}
			for _, r := range reqs {
				if r.Package == "Bob-Gone" {
					out[r.Package] = "bob-gone 9.9.9 is not in the index"
				}
			}
			return out
		},
	}), rec
}

func packMods(pv Preview) map[string]Mod {
	out := map[string]Mod{}
	for _, m := range pv.Mods {
		out[m.Package] = m
	}
	return out
}

func TestPreviewLinkReadsABareCodeAndAHashCode(t *testing.T) {
	s, _ := packService(t)
	for _, text := range []string{codeKey, "#r2modman\n" + base64.StdEncoding.EncodeToString(r2z(t))} {
		pv, err := s.PreviewLink(context.Background(), "lethal-company", text, "")
		if err != nil || pv.Name != "Just some mods" || pv.Game != "lethal-company" {
			t.Fatalf("preview %+v, %v", pv, err)
		}
		mods := packMods(pv)
		if mods["Alice-MoreCompany"].State != StateDownload || mods["Alice-MoreCompany"].Version != "1.2.3" {
			t.Errorf("alice = %+v", mods["Alice-MoreCompany"])
		}
		if gone := mods["Bob-Gone"]; gone.State != StateUnavailable || gone.Reason != ReasonNotInIndex || gone.Detail == "" {
			t.Errorf("an unresolvable package must be an unavailable row: %+v", gone)
		}
		// One config file lands. Mine's folder, which no package names, is a local mod whose DLL comes with it; a DLL
		// anywhere else is listed as left out.
		if pv.Settings != 1 || !slices.Equal(pv.SkippedPrograms, []string{"BepInEx/plugins/Alice-MoreCompany/extra.dll", "BepInEx/plugins/run.dll"}) {
			t.Errorf("settings %d, skipped %v", pv.Settings, pv.SkippedPrograms)
		}
		if mine := mods[""]; mine.Name != "Mine" || mine.Site != SiteLocal || mine.State != StateDownload || !mine.Enabled ||
			mine.FilesFrom != FilesFromCode || mine.SizeKB != 1 {
			t.Errorf("mine = %+v", mine)
		}
	}
}

// localEntry is the profile's local entry named name and the store folder holding its files.
func localEntry(t *testing.T, s *Service, profileID, name string) (profile.Entry, string) {
	t.Helper()
	p, err := s.find("lethal-company", profileID)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(p.Entries, func(e profile.Entry) bool { return e.Source.Kind == profile.KindLocal && e.Source.Name == name })
	if i < 0 {
		t.Fatalf("no local entry %s in %+v", name, p.Entries)
	}
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := items.Path("lethal-company", p.Entries[i].StoreKey())
	if err != nil {
		t.Fatal(err)
	}
	return p.Entries[i], dir
}

func TestImportFromACodeWritesConfigsAndQueuesWhatResolves(t *testing.T) {
	s, rec := packService(t)
	pv, err := s.PreviewLink(context.Background(), "lethal-company", codeKey, "")
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Import(context.Background(), "lethal-company", pv.Session, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.reqs) != 1 || rec.reqs[0].Package != "Alice-MoreCompany" {
		t.Fatalf("queued %+v", rec.reqs)
	}
	dir, err := s.d.Profiles.ProfileDir("lethal-company", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := fsx.ReadFile(filepath.Join(dir, "BepInEx", "config", "alice.cfg")); err != nil || string(got) != "[General]\n" {
		t.Fatalf("config = %q, %v", got, err)
	}
	e, files := localEntry(t, s, res.Profile.ID, "Mine")
	if len(e.Mods) != 1 || !e.Enabled(e.Mods[0].ID) || !fsx.IsFile(filepath.Join(files, "BepInEx", "plugins", "Mine.dll")) {
		t.Fatalf("Mine = %+v in %s", e, files)
	}
}

func TestImportFromAGaleProfileBringsItsLocalModAndItsDependencies(t *testing.T) {
	s, rec := packService(t)
	data := t.TempDir()
	prof := filepath.Join(data, "lethal-company", "profiles", "Crew")
	// Gale switches a mod off by adding .old to each of its files.
	testfs.WriteFile(t, prof, "BepInEx/plugins/Mine/Mine.dll.old", "MZ")
	testfs.WriteFile(t, prof, "BepInEx/plugins/Mine/data/a.bin.old", "x")
	db, err := sql.Open("sqlite", filepath.Join(data, "data.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE profiles (id INTEGER PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL, game_slug TEXT NOT NULL, mods JSON NOT NULL)`,
		`INSERT INTO profiles (name, path, game_slug, mods) VALUES ('Crew', '` + prof + `', 'lethal-company', '[
			{"enabled":true,"fullName":"Alice-MoreCompany-1.2.3"},
			{"enabled":false,"name":"Mine","author":"Me","version":"0.1.0","uuid":"x","fileSize":2,
			 "dependencies":["Alice-MoreCompany-1.2.3","Carol-Lib-2.0.0"]}]')`,
	} {
		if _, err := db.ExecContext(t.Context(), q); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s.d.PackFolder = func() string { return data }
	pv, err := s.PreviewLocalProfile(t.Context(), "lethal-company", prof, "")
	if err != nil {
		t.Fatal(err)
	}
	mods := packMods(pv)
	if mine := mods[""]; mine.Name != "Mine" || mine.Author != "Me" || mine.Version != "0.1.0" || mine.State != StateDownload ||
		mine.Enabled || mine.FilesFrom != FilesFromComputer {
		t.Errorf("mine = %+v", mine)
	}
	if mods["Carol-Lib"].State != StateDownload {
		t.Errorf("the local mod's dependency is not brought: %+v", pv.Mods)
	}
	res, err := s.Import(t.Context(), "lethal-company", pv.Session, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.reqs) != 2 {
		t.Errorf("queued %+v", rec.reqs)
	}
	e, files := localEntry(t, s, res.Profile.ID, "Mine")
	if len(e.Mods) != 1 || e.Enabled(e.Mods[0].ID) || !fsx.IsFile(filepath.Join(files, "BepInEx", "plugins", "Mine.dll")) ||
		!fsx.IsFile(filepath.Join(files, "BepInEx", "plugins", "data", "a.bin")) {
		t.Fatalf("Mine = %+v in %s", e, files)
	}
}

func TestPreviewFileReadsAnR2zAndALocalProfileFolder(t *testing.T) {
	s, _ := packService(t)
	file := filepath.Join(t.TempDir(), "friends.r2z")
	if err := os.WriteFile(file, r2z(t), 0o600); err != nil {
		t.Fatal(err)
	}
	pv, err := s.PreviewFile(context.Background(), "lethal-company", file, "")
	if err != nil || pv.Name != "Just some mods" || packMods(pv)["Alice-MoreCompany"].Key == "" {
		t.Fatalf("r2z preview %+v, %v", pv, err)
	}

	prof := filepath.Join(t.TempDir(), "LethalCompany", "profiles", "Mine")
	if err := os.MkdirAll(filepath.Join(prof, "BepInEx", "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prof, "mods.yml"), []byte("- name: Alice-MoreCompany\n  versionNumber: {major: 1, minor: 2, patch: 3}\n  enabled: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pv, err = s.PreviewLocalProfile(context.Background(), "lethal-company", prof, "")
	if err != nil || packMods(pv)["Alice-MoreCompany"].State != StateDownload {
		t.Fatalf("folder preview %+v, %v", pv, err)
	}
	res, err := s.Import(context.Background(), "lethal-company", pv.Session, "", nil)
	if err != nil || res.Profile.ImportedFrom != prof {
		t.Fatalf("imported from %q, %v", res.Profile.ImportedFrom, err)
	}
	if _, err := s.PreviewFile(context.Background(), "lethal-company", prof+"/mods.yml", ""); err == nil {
		t.Error("a file that is no pack previewed")
	}
}

func TestReplaceFromACodeRemovesThunderstoreModsItLacks(t *testing.T) {
	s, _ := packService(t)
	prof, err := s.d.Profiles.Create("lethal-company", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "x.zip"), map[string]string{"BepInEx/plugins/Extra/Extra.dll": "x"})
	if _, err := s.d.Profiles.InstallSource(t.Context(), "lethal-company", prof.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Old-Extra", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	pv, err := s.PreviewLink(context.Background(), "lethal-company", codeKey, prof.ID)
	if err != nil || len(pv.Replace.Remove) != 1 {
		t.Fatalf("plan %+v, %v", pv.Replace, err)
	}
	if _, err := s.Replace(context.Background(), "lethal-company", pv.Session, prof.ID, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := s.find("lethal-company", prof.ID)
	if slices.ContainsFunc(got.Entries, func(e profile.Entry) bool { return e.Source.Name == "Old-Extra" }) {
		t.Fatalf("Old-Extra still present: %+v", got.Entries)
	}
}

func TestImportFromAnR2modmanFolderBringsAHandAddedModAsLocal(t *testing.T) {
	s, _ := packService(t)
	prof := filepath.Join(t.TempDir(), "LethalCompany", "profiles", "Mine")
	// r2modman's Import local mod writes Author-Name to mods.yml like a package, and keeps its files under plugins.
	testfs.WriteFile(t, prof, "BepInEx/plugins/Bob-Gone/Gone.dll", "MZ")
	testfs.WriteFile(t, prof, "mods.yml", "- name: Alice-MoreCompany\n  versionNumber: {major: 1, minor: 2, patch: 3}\n  enabled: true\n"+
		"- name: Bob-Gone\n  versionNumber: {major: 9, minor: 9, patch: 9}\n  enabled: true\n")
	pv, err := s.PreviewLocalProfile(t.Context(), "lethal-company", prof, "")
	if err != nil {
		t.Fatal(err)
	}
	if gone := packMods(pv)["Bob-Gone"]; gone.Key != "" {
		t.Errorf("the folder's mod is still a Thunderstore row: %+v", gone)
	}
	i := slices.IndexFunc(pv.Mods, func(m Mod) bool { return m.Name == "Gone" })
	if i < 0 || pv.Mods[i].Site != SiteLocal || pv.Mods[i].FilesFrom != FilesFromComputer || pv.Mods[i].State != StateDownload {
		t.Fatalf("Gone is not a carried local mod: %+v", pv.Mods)
	}
	if slices.ContainsFunc(pv.Problems, func(p Problem) bool { return p.Name == "Gone" }) {
		t.Errorf("a problem is left for the adopted mod: %+v", pv.Problems)
	}
	res, err := s.Import(t.Context(), "lethal-company", pv.Session, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, files := localEntry(t, s, res.Profile.ID, "Gone"); !fsx.IsFile(filepath.Join(files, "BepInEx", "plugins", "Gone.dll")) {
		t.Errorf("the local mod's files are not in %s", files)
	}
}

func TestAPackageTheIndexLacksIsNotRemovedFromTheSite(t *testing.T) {
	s, _ := packService(t)
	pv, err := s.PreviewLink(context.Background(), "lethal-company", codeKey, "")
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(pv.Problems, func(p Problem) bool { return p.Name == "Gone" })
	if i < 0 {
		t.Fatalf("no problem for Gone: %+v", pv.Problems)
	}
	if p := pv.Problems[i]; p.Kind != ProblemNotOnSource || p.Site != SiteThunderstore || p.URL != "" {
		t.Errorf("problem = %+v", p)
	}
}

func TestImportKeepsAProfilesOwnConfigUnlessReplaceIsChosen(t *testing.T) {
	s, _ := packService(t)
	prof, err := s.d.Profiles.Create("lethal-company", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.d.Profiles.ProfileDir("lethal-company", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "BepInEx", "config", "alice.cfg")
	testfs.WriteAt(t, cfg, []byte("mine\n"))
	read := func() string {
		b, err := os.ReadFile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	pv, err := s.PreviewLink(t.Context(), "lethal-company", codeKey, prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(t.Context(), "lethal-company", pv.Session, prof.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got := read(); got != "mine\n" {
		t.Fatalf("an import into the profile replaced its own config: %q", got)
	}
	pv, err = s.PreviewLink(t.Context(), "lethal-company", codeKey, prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Replace(t.Context(), "lethal-company", pv.Session, prof.ID, nil); err != nil {
		t.Fatal(err)
	}
	if got := read(); got != "[General]\n" {
		t.Fatalf("a Replace kept the profile's config: %q", got)
	}
}
