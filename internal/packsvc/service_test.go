package packsvc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	drafts   []profile.ModpackDraft
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

func (f *fakeProfiles) SetImportedFrom(_, id, path string) (profile.Profile, error) {
	p := profile.Profile{ID: id, ImportedFrom: path}
	f.list = append(f.list, p)
	return p, nil
}

func (f *fakeProfiles) SetModpackDraft(_, id string, d profile.ModpackDraft) (profile.Profile, error) {
	f.drafts = append(f.drafts, d)
	for i := range f.list {
		if f.list[i].ID == id {
			f.list[i].Modpack = &d
		}
	}
	return profile.Profile{}, nil
}

func (f *fakeProfiles) ProfileDir(string, string) (string, error) { return f.dir, nil }

func (*fakeProfiles) Backup(string, string) (profile.Profile, map[string][]byte, error) {
	return profile.Profile{}, nil, errors.New("not in this fake")
}

func (*fakeProfiles) BackupDirs(string, profile.Profile, func(profile.Entry) bool) (map[string]string, error) {
	return nil, errors.New("not in this fake")
}

func (*fakeProfiles) RestoreBackup(context.Context, string, profile.Profile, map[string][]byte, map[string]string) (profile.Profile, []profile.Entry, error) {
	return profile.Profile{}, nil, errors.New("not in this fake")
}

type fakeQueue struct {
	got []queue.Request
	bad map[string]string
}

func (f *fakeQueue) Unresolvable(context.Context, string, string, []queue.Request) map[string]string {
	return f.bad
}

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
	store := &fakeProfiles{}
	if _, err := store.SetImportedFrom("lethal-company", "p1", filepath.Join(profiles, "Friends")); err != nil {
		t.Fatal(err)
	}
	if got, _ := (&Service{Profiles: store}).LocalProfiles("lethal-company"); len(got) != 1 || !got[0].Imported {
		t.Fatalf("an imported profile is not marked: %+v", got)
	}
}

func TestLocalProfilesReadsTheFolderThePlayerChose(t *testing.T) {
	empty := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", empty)
	t.Setenv("XDG_DATA_HOME", empty)
	t.Setenv("APPDATA", empty)
	chosen := t.TempDir()
	profile := filepath.Join(chosen, "LethalCompany", "profiles", "Crew")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "mods.yml"), []byte("- name: Ns-Mod\n- name: Ns-Two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, _ := (&Service{}).LocalProfiles("lethal-company"); len(got) != 0 {
		t.Fatalf("a folder nobody chose was read: %+v", got)
	}
	got, err := (&Service{Folder: func() string { return chosen }}).LocalProfiles("lethal-company")
	if err != nil || len(got) != 1 || got[0].Name != "Crew" || got[0].Path != profile || got[0].Mods != 2 {
		t.Fatalf("LocalProfiles = %+v, %v", got, err)
	}
}

func TestExportCodeCarriesTheConfigFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "BepInEx", "config", "Sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "BepInEx", "config", "Sub", "a.cfg"), []byte("x=1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "BepInEx", "core"), 0o750); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"BepInEx/core/BepInEx.Preloader.dll": "p", ".mortar-bepinex.json": `{"version":"5.4.2305","doorstop":4}`} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{"key":"k"}`))
	}))
	defer srv.Close()
	ps := &fakeProfiles{dir: dir, list: []profile.Profile{{ID: "p1", Name: "P", Entries: []profile.Entry{
		{Source: profile.Source{Kind: profile.KindThunderstore, Name: "A-B", Version: "1.0.0"}},
	}}}}
	if _, err := (&Service{Profiles: ps, Code: pack.Code{URL: srv.URL}}).ExportCode(context.Background(), "lethal-company", "p1", true); err != nil {
		t.Fatal(err)
	}
	_, enc, _ := strings.Cut(body, "\n")
	z, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(z), int64(len(z)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	if !slices.Contains(names, "config/Sub/a.cfg") {
		t.Errorf("code holds %v, not the profile's config file", names)
	}
	r2x, err := fs.ReadFile(zr, "export.r2x")
	if err != nil || !strings.Contains(string(r2x), "name: BepInEx-BepInExPack") || !strings.Contains(string(r2x), "patch: 2305") {
		t.Errorf("r2modman installs only what a code lists, so it must list the profile's BepInEx pack: %s %v", r2x, err)
	}
}

func TestImportAsNamesTheNewProfile(t *testing.T) {
	ps, q := &fakeProfiles{}, &fakeQueue{}
	s := &Service{Profiles: ps, Queue: q}
	if _, err := s.ImportAs(context.Background(), Source{Path: writeR2z(t)}, "stardew", "  Crew night "); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportAs(context.Background(), Source{Path: writeR2z(t)}, "stardew", " "); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ps.created, []string{"Crew night", "Friends"}) {
		t.Errorf("created %v", ps.created)
	}
}

func TestUpdatingAProfileQueuesIntoItAndExtrasListsWhatThePackLacks(t *testing.T) {
	ts := func(key, name string, implicit bool) profile.Entry {
		return profile.Entry{Key: key, Package: true, Implicit: implicit, Source: profile.Source{Kind: profile.KindThunderstore, Name: name, Version: "1.0.0"}}
	}
	target := profile.Profile{ID: "p9", Entries: []profile.Entry{
		ts("k1", "alice-morecompany", false), ts("k2", "Bob-Cheaty", false), ts("k3", "Carol-Lib", true),
		{Key: "k4", Source: profile.Source{Kind: profile.KindNexus, Name: "Nexus mod"}},
	}}
	ps, q := &fakeProfiles{list: []profile.Profile{target}}, &fakeQueue{}
	s := &Service{Profiles: ps, Queue: q}
	src := Source{Path: writeR2z(t)}
	res, err := s.Import(context.Background(), src, "stardew", "p9")
	if err != nil || res.Profile != "p9" || len(ps.created) != 0 || q.got[0].Profile != "p9" {
		t.Fatalf("update %+v %v created %v queued %+v", res, err, ps.created, q.got)
	}
	extras, err := s.Extras(context.Background(), src, "stardew", "p9")
	if err != nil || len(extras) != 1 || extras[0].Key != "k2" {
		t.Fatalf("a package the pack lacks is offered, a listed one (any case), a dependency and a non-Thunderstore entry are not: %+v %v", extras, err)
	}
	if _, err := s.Extras(context.Background(), src, "stardew", "nope"); err == nil {
		t.Error("extras of a missing profile")
	}
}

func TestMirrorClientQueuesTheClientsPackagesIntoTheServerProfile(t *testing.T) {
	ps, q := &fakeProfiles{dir: t.TempDir(), list: []profile.Profile{{ID: "c1", Name: "Crew", Entries: []profile.Entry{
		{Source: profile.Source{Kind: profile.KindThunderstore, Name: "Alice-Mod", Version: "1.0.0"}},
		{Source: profile.Source{Kind: profile.KindThunderstore, Name: "denikson-BepInExPack_Valheim", Version: "5.4.2202"}},
		{Source: profile.Source{Kind: profile.KindNexus, ModID: 5}},
	}}}}, &fakeQueue{}
	s := &Service{Profiles: ps, Queue: q}
	res, err := s.MirrorClient(context.Background(), "valheim-server", "srv", "c1")
	if err != nil || res.Queued != 1 {
		t.Fatalf("MirrorClient = %+v, %v", res, err)
	}
	if len(q.got) != 1 || q.got[0].Game != "valheim-server" || q.got[0].Profile != "srv" || q.got[0].Package != "Alice-Mod" {
		t.Fatalf("queued %+v", q.got)
	}
	if _, err := s.MirrorClient(context.Background(), "valheim", "srv", "c1"); err == nil {
		t.Fatal("a client game was treated as a server")
	}
}

func TestServerSettingsFormKeepsSecretsInsideMortar(t *testing.T) {
	ps := &fakeProfiles{dir: t.TempDir()}
	s := &Service{Profiles: ps}
	if err := s.SetServerSetting("valheim-server", "srv", "password", "abc"); err == nil {
		t.Fatal("a password shorter than the catalog's pattern was saved")
	}
	for id, v := range map[string]string{"password": "hunter22", "world": "Mine"} {
		if err := s.SetServerSetting("valheim-server", "srv", id, v); err != nil {
			t.Fatal(err)
		}
	}
	form, err := s.ServerSettings("valheim-server", "srv")
	if err != nil || form.Clients != "valheim" || form.ClientName != "Valheim" {
		t.Fatalf("form = %+v, %v", form, err)
	}
	for _, set := range form.Settings {
		switch set.ID {
		case "password":
			if set.Value != "" || !set.Set || !set.Secret || !set.Required {
				t.Errorf("password = %+v", set)
			}
		case "world":
			if set.Value != "Mine" || set.Default != "Dedicated" {
				t.Errorf("world = %+v", set)
			}
		}
	}
	if plain, err := s.ServerSettings("valheim", "p"); err != nil || plain.Clients != "" || len(plain.Settings) != 0 {
		t.Fatalf("client game form = %+v, %v", plain, err)
	}
}

func TestImportOverAProfileKeepsDifferingFilesUnlessReplaceIsChosen(t *testing.T) {
	base := t.TempDir()
	ps := profile.OpenIn(filepath.Join(base, "profiles"), store.OpenAt(filepath.Join(base, "store")))
	target, err := ps.Create("lethal-company", "Mine")
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := ps.ProfileDir("lethal-company", target.ID)
	cfg := filepath.Join(dir, "BepInEx", "plugins", "a.txt")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"export.r2x":            "profileName: Friends\nmods:\n  - name: BepInEx-BepInExPack\n    version: {major: 5, minor: 4, patch: 2100}\n    enabled: true\n",
		"bepinex/plugins/a.txt": "pack",
		"bepinex/plugins/b.txt": "new",
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
	s := &Service{Profiles: ps, Queue: &fakeQueue{}}
	res, err := s.Import(context.Background(), Source{Path: path}, "lethal-company", target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(cfg); string(b) != "mine" || !slices.Equal(res.Kept, []string{"BepInEx/plugins/a.txt"}) {
		t.Fatalf("the player's file was replaced or not reported: %q, kept %v", b, res.Kept)
	}
	if b, _ := fsx.ReadFile(filepath.Join(dir, "BepInEx", "plugins", "b.txt")); string(b) != "new" {
		t.Errorf("a file the profile lacked was not written: %q", b)
	}
	if _, err := s.Import(context.Background(), Source{Path: path, Replace: true}, "lethal-company", target.ID); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(cfg); string(b) != "pack" {
		t.Errorf("Replace did not write the pack's file: %q", b)
	}
}

func TestImportSkipsUnresolvablePackagesAndKeepsTheRest(t *testing.T) {
	q := &fakeQueue{bad: map[string]string{"Alice-MoreCompany": "alice-morecompany 1.2.3 is not in the index"}}
	res, err := (&Service{Profiles: &fakeProfiles{}, Queue: q}).Import(context.Background(), Source{Path: writeR2z(t)}, "stardew", "")
	if err != nil || res.Queued != 1 || len(q.got) != 1 || q.got[0].Package != "BepInEx-BepInExPack" {
		t.Fatalf("queued %+v, %+v, %v", q.got, res, err)
	}
	if !slices.Equal(res.Skipped, []string{"Alice-MoreCompany: alice-morecompany 1.2.3 is not in the index"}) || len(res.Disabled) != 0 {
		t.Errorf("result %+v", res)
	}
}
