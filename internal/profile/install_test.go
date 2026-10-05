package profile

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/github"
)

func buildZip(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	return testfs.WriteZip(t, filepath.Join(t.TempDir(), name), files)
}

func TestInstallArchive(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	good := buildZip(t, "Pack.zip", map[string]string{"Pack/A/manifest.json": manifestJSON("X.A"), "Pack/B/manifest.json": manifestJSON("X.B")})

	res, err := e.InstallArchive("stardew", p.ID, good)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 2 || res.Added[0] != "X.A" || res.Profile.Entries[0].Source != (Source{Kind: KindLocal, Name: "Pack.zip"}) {
		t.Fatalf("result = %+v", res)
	}

	_, err = e.InstallArchive("stardew", p.ID, good)
	var ie *InstallError
	if !errors.As(err, &ie) || ie.Msg != "X.A, X.B is already in this profile" || !errors.As(err, new(*DuplicateError)) {
		t.Fatalf("duplicate err = %v", err)
	}

	empty := buildZip(t, "Empty.zip", map[string]string{"readme.txt": "x"})
	_, err = e.InstallArchive("stardew", p.ID, empty)
	if !errors.As(err, &ie) || ie.Msg != "No SMAPI mod was found in this archive" {
		t.Fatalf("loose files err = %v", err)
	}

	txt := filepath.Join(t.TempDir(), "notes.txt")
	if err := fsx.WriteFile(txt, []byte("not an archive at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = e.InstallArchive("stardew", p.ID, txt)
	if !errors.As(err, &ie) || !strings.HasPrefix(ie.Msg, "Mortar reads zip") || !errors.Is(err, archive.ErrUnsupportedFormat) {
		t.Fatalf("unsupported err = %v", err)
	}

	evil := buildZip(t, "Evil.zip", map[string]string{"../x/manifest.json": manifestJSON("X.E")})
	_, err = e.InstallArchive("stardew", p.ID, evil)
	if !errors.As(err, &ie) || !errors.Is(err, archive.ErrTraversal) {
		t.Fatalf("unsafe err = %v", err)
	}
}

func TestInstallArchiveRawXNBExplainsContentPatcherAlternative(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	raw := buildZip(t, "raw-woods.zip", map[string]string{"Woods.xnb": "raw"})

	_, err := e.InstallArchive("stardew", p.ID, raw)
	var ie *InstallError
	if !errors.As(err, &ie) || ie.Msg != "This file replaces game files directly (raw .xnb). Mortar installs SMAPI mods; use the mod's Content Patcher version." {
		t.Fatalf("raw xnb err = %v", err)
	}
}

func TestInstallArchiveSameModVersionIsAReplace(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	v1 := buildZip(t, "a-1.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	res, err := e.InstallArchive("stardew", p.ID, v1)
	if err != nil || res.Updated || res.VersionChanged {
		t.Fatalf("first %+v %v", res, err)
	}
	v2 := buildZip(t, "a-2.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "A/extra.txt": "x"})
	res, err = e.InstallArchive("stardew", p.ID, v2)
	if err != nil || !res.Updated || res.VersionChanged {
		t.Fatalf("same version %+v %v", res, err)
	}
}

func TestInstallArchiveNewModVersionIsAnUpdate(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	v1 := buildZip(t, "a-1.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	if _, err := e.InstallArchive("stardew", p.ID, v1); err != nil {
		t.Fatal(err)
	}
	v2 := buildZip(t, "a-2.zip", map[string]string{"A/manifest.json": `{"Name":"X.A","Author":"me","Version":"2.0.0","UniqueID":"X.A"}`})
	res, err := e.InstallArchive("stardew", p.ID, v2)
	if err != nil || !res.Updated || !res.VersionChanged {
		t.Fatalf("new version %+v %v", res, err)
	}
}

func TestInstallNexusKeepsTheSourceAndUpdatesInPlace(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	v1 := buildZip(t, "a-1.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	src := Source{Kind: KindNexus, Name: "a-1.zip", ModID: 7, FileID: 1, Version: "1.0", Picture: "https://img/a.png", EndorsementCount: 3}
	res, err := e.InstallNexus("stardew", p.ID, v1, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || res.Profile.Entries[0].Key != "nexus-7-1" || res.Profile.Entries[0].Source != src {
		t.Fatalf("first install: %+v", res)
	}
	v2 := buildZip(t, "a-2.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "A/extra.txt": "new"})
	next := Source{Kind: KindNexus, Name: "a-2.zip", ModID: 7, FileID: 2, Version: "2.0"}
	res, err = e.InstallNexus("stardew", p.ID, v2, next)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || len(res.Profile.Entries) != 1 || res.Profile.Entries[0].Key != "nexus-7-2" || res.Profile.Entries[0].PreviousKey != "nexus-7-1" || res.Profile.Entries[0].Source != next {
		t.Fatalf("update: %+v", res)
	}
	mods, err := e.UserMods("stardew", p.ID)
	if err != nil || len(mods) != 1 || mods[0].Picture != "" {
		t.Fatalf("mods %+v, %v", mods, err)
	}
}

func TestStageThenInstallGitHubKeepsTheTypedSource(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	zip := buildZip(t, "mod.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	src := Source{Kind: KindGitHub, Name: "mod.zip", Version: "1.0", Repo: "Me/Mod", Tag: "v1.0", Asset: "mod.zip"}
	key, ids, err := e.StageGitHub("stardew", src, zip)
	if err != nil || key != github.Key("Me", "Mod", "v1.0", "mod.zip") || len(ids) != 1 || ids[0] != "smapi:X.A" {
		t.Fatalf("stage = %q, %v, %v", key, ids, err)
	}
	if got, _ := e.List("stardew"); len(got[0].Entries) != 0 {
		t.Fatal("staging changed the profile")
	}
	if _, err := e.InstallStaged("stardew", p.ID, key, src); err != nil {
		t.Fatal(err)
	}
	got, err := e.List("stardew")
	if err != nil || got[0].Entries[0].Source != src {
		t.Fatalf("source after a round trip through profile.json: %+v, %v", got, err)
	}
}

func TestConcurrentInstallsOfOneModKeepOneEntry(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	a := buildZip(t, "A.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "A/one.txt": "1"})
	b := buildZip(t, "B.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "A/two.txt": "2"})

	errs := make(chan error, 2)
	for _, path := range []string{a, b} {
		go func() {
			_, err := e.InstallArchive("stardew", p.ID, path)
			errs <- err
		}()
	}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	got, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %+v", got.Entries)
	}
}
