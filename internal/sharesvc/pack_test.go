package sharesvc

import (
	"archive/zip"
	"bytes"
	"context"
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
	for name, body := range map[string]string{"export.r2x": packExport, "config/alice.cfg": "[General]\n", "BepInEx/plugins/run.dll": "MZ"} {
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
		// One config file lands; run.dll is an executable type and is listed as left out.
		if pv.Settings != 1 || !slices.Equal(pv.SkippedPrograms, []string{"BepInEx/plugins/run.dll"}) {
			t.Errorf("settings %d, skipped %v", pv.Settings, pv.SkippedPrograms)
		}
	}
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
