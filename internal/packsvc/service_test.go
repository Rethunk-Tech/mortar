package packsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/pack"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/store"

	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

type fakeProfiles struct {
	created  []string
	written  map[string][]byte
	writeErr error
	list     []profile.Profile
	dir      string
}

func (f *fakeProfiles) WriteFiles(_, _ string, files map[string][]byte) error {
	f.written = files
	return f.writeErr
}

func (f *fakeProfiles) Create(_, name string) (profile.Profile, error) {
	f.created = append(f.created, name)
	return profile.Profile{ID: "p1", Name: name}, nil
}

func (f *fakeProfiles) List(string) ([]profile.Profile, error) { return f.list, nil }

func (f *fakeProfiles) ProfileDir(string, string) (string, error) { return f.dir, nil }

func (*fakeProfiles) Backup(string, string) (profile.Profile, map[string][]byte, error) {
	return profile.Profile{}, nil, errors.New("not in this fake")
}

func (*fakeProfiles) BackupDirs(string, profile.Profile, func(profile.Entry) bool) (map[string]string, error) {
	return nil, errors.New("not in this fake")
}

func (*fakeProfiles) RestoreBackup(string, profile.Profile, map[string][]byte, map[string]string) (profile.Profile, []profile.Entry, error) {
	return profile.Profile{}, nil, errors.New("not in this fake")
}

type fakeQueue struct{ got []queue.Request }

func (f *fakeQueue) Add(_ context.Context, reqs []queue.Request) ([]queue.Item, error) {
	f.got = append(f.got, reqs...)
	return make([]queue.Item, len(reqs)), nil
}

func writeR2z(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("export.r2x")
	_, _ = w.Write([]byte(`profileName: Friends
mods:
  - name: BepInEx-BepInExPack
    version: {major: 5, minor: 4, patch: 2100}
    enabled: true
  - name: Alice-MoreCompany
    version: {major: 1, minor: 2, patch: 3}
    enabled: false
`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "friends.r2z")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportQueuesThePacksPackagesIntoANewProfile(t *testing.T) {
	ps, q := &fakeProfiles{}, &fakeQueue{}
	s := &Service{Profiles: ps, Queue: q}
	src := Source{Path: writeR2z(t)}
	pv, err := s.Preview(context.Background(), src)
	if err != nil || pv.Name != "Friends" || len(pv.Packages) != 2 {
		t.Fatalf("preview %+v, %v", pv, err)
	}
	if _, err := s.Import(context.Background(), src, "", ""); err == nil {
		t.Error("a pack without a game imported")
	}
	res, err := s.Import(context.Background(), src, "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Queued != 2 || res.Profile != "p1" || !slices.Equal(res.Disabled, []string{"Alice-MoreCompany"}) || !slices.Equal(ps.created, []string{"Friends"}) {
		t.Errorf("result %+v, created %v", res, ps.created)
	}
	if q.got[1].Disabled == nil || q.got[0].Disabled != nil {
		t.Errorf("disabled flags %+v", q.got)
	}
	if len(q.got) != 2 || q.got[1].Package != "Alice-MoreCompany" || q.got[1].Version != "1.2.3" || q.got[1].Profile != "p1" || q.got[1].Game != "stardew" {
		t.Errorf("queued %+v", q.got)
	}
}

func TestImportQueuesNothingWhenThePacksFilesCannotBeWritten(t *testing.T) {
	ps, q := &fakeProfiles{writeErr: errors.New("disk full")}, &fakeQueue{}
	if _, err := (&Service{Profiles: ps, Queue: q}).Import(context.Background(), Source{Path: writeR2z(t)}, "stardew", "p1"); err == nil {
		t.Fatal("a failed write reported success")
	}
	if len(q.got) != 0 || len(ps.created) != 0 {
		t.Fatalf("queued %+v, created %v after a failed write", q.got, ps.created)
	}
}

func TestExportNeedsTheConfirmation(t *testing.T) {
	if _, err := (&Service{}).ExportCode(context.Background(), "stardew", "p1", false); err == nil {
		t.Error("an unconfirmed export ran")
	}
}

func TestPackFilesLandUnderBepInEx(t *testing.T) {
	got := packFiles(pack.Draft{
		Loose:   []pack.File{{Path: "BepInEx/plugins/a.txt", Data: []byte("a")}, {Path: "BepInEx/plugins/b.dll", Data: []byte("b")}},
		Configs: []pack.File{{Path: "config/x.cfg", Data: []byte("x")}},
	}, importRoots("lethal-company"))
	if string(got["BepInEx/config/x.cfg"]) != "x" || string(got["BepInEx/plugins/a.txt"]) != "a" || len(got) != 2 {
		t.Fatalf("files = %v", got)
	}
}

func TestPackFilesJudgeNamesAsWindowsStoresThem(t *testing.T) {
	got := packFiles(pack.Draft{Loose: []pack.File{
		{Path: "BepInEx/plugins/evil.dll.", Data: []byte("MZ")},
		{Path: "BepInEx/plugins/evil2.dll ", Data: []byte("MZ")},
		{Path: "BepInEx/plugins/. ./x.txt", Data: []byte("x")},
		{Path: "BepInEx/plugins/CON", Data: []byte("x")},
		{Path: "BepInEx/plugins/notes.txt. ", Data: []byte("n")},
	}}, importRoots("lethal-company"))
	if len(got) != 1 || string(got["BepInEx/plugins/notes.txt"]) != "n" {
		t.Fatalf("files = %v", slices.Collect(maps.Keys(got)))
	}
}

func TestImportWritesNothingOutsideTheLoaderFolder(t *testing.T) {
	base := t.TempDir()
	ps := profile.OpenIn(filepath.Join(base, "profiles"), store.OpenAt(filepath.Join(base, "store")))
	target, err := ps.Create("lethal-company", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"export.r2x":            "profileName: Friends\nmods:\n  - name: BepInEx-BepInExPack\n    version: {major: 5, minor: 4, patch: 2100}\n    enabled: true\n",
		"profile.json":          `{"id":"zzz","name":"Mine","entries":[],"launchPrefix":"sh -c 'touch /tmp/pwned' --"}`,
		"Profile.JSON":          `{}`,
		"saves/slot":            "x",
		"bepinex/plugins/a.txt": "a",
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "friends.r2z")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Service{Profiles: ps, Queue: &fakeQueue{}}).Import(context.Background(), Source{Path: path}, "lethal-company", target.ID); err != nil {
		t.Fatal(err)
	}
	all, err := ps.List("lethal-company")
	if err != nil || len(all) != 1 || all[0].ID != target.ID || all[0].LaunchPrefix != "" {
		t.Fatalf("the pack rewrote the profile: %+v, %v", all, err)
	}
	dir, _ := ps.ProfileDir("lethal-company", target.ID)
	if _, err := os.Stat(filepath.Join(dir, "saves", "slot")); err == nil {
		t.Error("the pack wrote into the profile's saves")
	}
	if b, err := fsx.ReadFile(filepath.Join(dir, "BepInEx", "plugins", "a.txt")); err != nil || string(b) != "a" {
		t.Errorf("a file under the loader folder was not written under its spelling: %q, %v", b, err)
	}
}

func TestLocalProfilesListsTheGamesR2modmanProfiles(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("APPDATA", cfg)
	profiles := filepath.Join(cfg, "r2modmanPlus-local", "LethalCompany", "profiles")
	if err := os.MkdirAll(filepath.Join(profiles, "Friends"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profiles, "Friends", "mods.yml"), []byte("- name: Ns-Mod\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(profiles, "Empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := (&Service{}).LocalProfiles("lethal-company")
	if err != nil || len(got) != 1 || got[0].Name != "Friends" || got[0].Path != filepath.Join(profiles, "Friends") || got[0].Source != "r2modman" || got[0].Mods != 1 {
		t.Fatalf("LocalProfiles = %+v, %v", got, err)
	}
	if got, _ := (&Service{}).LocalProfiles("stardew"); len(got) != 0 {
		t.Fatalf("a game r2modman does not know: %+v", got)
	}
}
