package profile

import (
	"archive/zip"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/archive"
)

func buildZip(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	f, err := fsx.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for n, body := range files {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := errors.Join(zw.Close(), f.Close()); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInstallArchive(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	good := buildZip(t, "Pack.zip", map[string]string{"Pack/A/manifest.json": manifestJSON("X.A"), "Pack/B/manifest.json": manifestJSON("X.B")})

	res, err := e.InstallArchive("stardew", p.ID, good)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 2 || res.Added[0] != "X.A" || res.Profile.Entries[0].Source != (Source{Kind: "local", Name: "Pack.zip"}) {
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
		t.Fatalf("no manifest err = %v", err)
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

func TestInstallNexusKeepsTheSourceAndUpdatesInPlace(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	v1 := buildZip(t, "a-1.zip", map[string]string{"A/manifest.json": manifestJSON("X.A")})
	src := Source{Kind: "nexus", Name: "a-1.zip", ModID: 7, FileID: 1, Version: "1.0", Picture: "https://img/a.png", EndorsementCount: 3}
	res, err := e.InstallNexus("stardew", p.ID, v1, src)
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || res.Profile.Entries[0].Key != "nexus-7-1" || res.Profile.Entries[0].Source != src {
		t.Fatalf("first install: %+v", res)
	}
	v2 := buildZip(t, "a-2.zip", map[string]string{"A/manifest.json": manifestJSON("X.A"), "A/extra.txt": "new"})
	next := Source{Kind: "nexus", Name: "a-2.zip", ModID: 7, FileID: 2, Version: "2.0"}
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
