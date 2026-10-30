package devsvc

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const adoptGame = "lethal-company"

func adoptFolder(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for f, body := range files {
		if err := writeAll(filepath.Join(dir, f), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDetectPicksTheKindFromWhatTheFolderHolds(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	cases := []struct {
		name, folder string
		files        map[string]string
		kind, csproj string
	}{
		{"csproj outside src", "Wallet", map[string]string{"lib/Other.csproj": "<Project/>", "Wallet/Wallet.csproj": "<Project/>", "a.txt": "x"}, "code", "Wallet/Wallet.csproj"},
		{"shallowest csproj", "Folder", map[string]string{"a/b/Deep.csproj": "<Project/>", "a/Top.csproj": "<Project/>"}, "code", "a/Top.csproj"},
		{"asset project", "Quacks", map[string]string{"plugins/sounds/quack.ogg": "ogg"}, "sounds", ""},
		{"bundle only", "Wallet-OuterWildsItems", map[string]string{"wallet.bundle": "b", "README.md": "r"}, "generic", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, err := svc.Detect(adoptGame, adoptFolder(t, c.folder, c.files))
			if err != nil || a.Known || a.Kind != c.kind || a.Csproj != c.csproj || !slices.Contains(a.Kinds, "generic") {
				t.Fatalf("detect = %+v, %v", a, err)
			}
		})
	}
	if _, err := svc.Detect(adoptGame, adoptFolder(t, "Empty", map[string]string{".git/HEAD": "ref"})); err == nil {
		t.Error("a folder with no files was offered")
	}
}

func TestAdoptingABundleOnlyFolderWritesOnlyItsRecordAndInstallsAsGeneric(t *testing.T) {
	svc, d := testService(t, testTemplate)
	dir := adoptFolder(t, "Wallet-OuterWildsItems", map[string]string{"wallet.bundle": "b", "Thumbs.db": "junk"})
	a, err := svc.Detect(adoptGame, dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := svc.Adopt(t.Context(), adoptGame, dir, "", a)
	if err != nil || p.Kind != "generic" || p.Name != "WalletOuterWildsItems" || !p.Untrusted {
		t.Fatalf("adopt = %+v, %v", p, err)
	}
	var written []string
	_ = filepath.WalkDir(dir, func(path string, e os.DirEntry, _ error) error {
		if rel, _ := filepath.Rel(dir, path); !e.IsDir() {
			written = append(written, filepath.ToSlash(rel))
		}
		return nil
	})
	slices.Sort(written)
	if !slices.Equal(written, []string{".mortar/project.json", "Thumbs.db", "wallet.bundle"}) {
		t.Errorf("folder holds %v", written)
	}
	if err := svc.Trust(adoptGame, dir); err != nil {
		t.Fatal(err)
	}
	if got := svc.Projects(adoptGame)[0]; got.Output != dir {
		t.Errorf("trusted output %q", got.Output)
	}
	if res, err := svc.Build(t.Context(), adoptGame, dir); err != nil || !res.OK || slices.ContainsFunc(d.calls, func(c []string) bool { return c[0] == "dotnet" }) {
		t.Errorf("build = %+v, %v, calls %v", res, err, d.calls)
	}
	info, err := svc.Inspect(t.Context(), adoptGame, dir)
	if err != nil || info.Kind != "generic" || info.Missing {
		t.Fatalf("info = %+v, %v", info, err)
	}
	r, err := svc.Pack(t.Context(), adoptGame, dir)
	if err != nil || !r.OK {
		t.Fatalf("pack = %+v, %v", r, err)
	}
	zr, err := zip.OpenReader(r.Zip)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"manifest.json", "wallet.bundle"}) {
		t.Errorf("zip holds %v", names)
	}
	if m := zipText(t, zr, "manifest.json"); !strings.Contains(m, `"name": "WalletOuterWildsItems"`) || !strings.Contains(m, `"version_number": "0.1.0"`) {
		t.Errorf("manifest %s", m)
	}
}

func TestAdoptedCodeProjectBuildsItsRecordedProjectFile(t *testing.T) {
	svc, d := testService(t, testTemplate)
	dir := adoptFolder(t, "Wallet", map[string]string{"Wallet/Wallet.csproj": "<Project/>"})
	a, err := svc.Detect(adoptGame, dir)
	if err != nil || a.Kind != "code" {
		t.Fatalf("detect = %+v, %v", a, err)
	}
	p, err := svc.Adopt(t.Context(), adoptGame, dir, "", a)
	if err != nil || p.Kind != "" || p.Name != "Wallet" {
		t.Fatalf("adopt = %+v, %v", p, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Config.Build.user.props")); err == nil {
		t.Error("adopting wrote a props file into the folder")
	}
	if err := svc.Trust(adoptGame, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Build(t.Context(), adoptGame, dir); err != nil {
		t.Fatal(err)
	}
	last := d.calls[len(d.calls)-1]
	if last[0] != "dotnet" || !slices.Contains(last, filepath.Join(dir, "Wallet", "Wallet.csproj")) || !slices.Contains(last, "-o") {
		t.Errorf("build ran %v", last)
	}
	if info, err := svc.Inspect(t.Context(), adoptGame, dir); err != nil || info.Missing || info.Name != "Wallet" {
		t.Errorf("info = %+v, %v", info, err)
	}
}

func TestRecordsRoundTripAndOldOnesStillRead(t *testing.T) {
	dir := assetDir(t, `{"kind":"sounds","name":"Quacks","author":"Me","team":"Ab"}`)
	if rec, ok := readRecord(dir); !ok || rec.Kind != "sounds" || rec.Csproj != "" {
		t.Fatalf("old record = %+v, %v", rec, ok)
	}
	for _, rec := range []assetRecord{{Kind: "generic", Name: "Bundle"}, {Kind: "code", Name: "My.Mod", Csproj: "a/My.Mod.csproj"}} {
		dir := t.TempDir()
		body := `{"kind":"` + rec.Kind + `","name":"` + rec.Name + `","csproj":"` + rec.Csproj + `"}`
		if err := writeAll(filepath.Join(dir, ".mortar", assetRecordFile), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if got, ok := readRecord(dir); !ok || got != rec {
			t.Errorf("record %+v read as %+v, %v", rec, got, ok)
		}
	}
	bad := t.TempDir()
	if err := writeAll(filepath.Join(bad, ".mortar", assetRecordFile), []byte(`{"kind":"code","name":"X1","csproj":"../out/X.csproj"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := readRecord(bad); ok {
		t.Error("a record escaped the project folder")
	}
}

func TestAdoptedCodeProjectReleasePacksItsBuiltDLL(t *testing.T) {
	svc, _ := testService(t, testTemplate)
	dir := adoptFolder(t, "Wallet", map[string]string{
		"Wallet/Wallet.csproj": "<Project><PropertyGroup><Version>1.2.3.0</Version></PropertyGroup></Project>",
		"README.md":            "# Wallet",
	})
	a, err := svc.Detect(adoptGame, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Adopt(t.Context(), adoptGame, dir, "", a); err != nil {
		t.Fatal(err)
	}
	if err := svc.Trust(adoptGame, dir); err != nil {
		t.Fatal(err)
	}
	svc.Run = func(_ context.Context, _, name string, args ...string) ([]byte, error) {
		if name != "dotnet" {
			return nil, errors.New("no tag")
		}
		out := args[slices.Index(args, "-o")+1]
		for _, f := range []string{"Wallet.dll", "Wallet.pdb", "UnityEngine.CoreModule.dll", "System.Memory.dll", "Helper.dll"} {
			if err := writeAll(filepath.Join(out, f), []byte(f), 0o644); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	r, err := svc.Pack(t.Context(), adoptGame, dir)
	if err != nil || !r.OK || r.Zip == "" {
		t.Fatalf("pack = %+v, %v", r, err)
	}
	zr, err := zip.OpenReader(r.Zip)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Helper.dll", "README.md", "Wallet.dll", "manifest.json"}) {
		t.Errorf("zip holds %v", names)
	}
	if m := zipText(t, zr, "manifest.json"); !strings.Contains(m, `"name": "Wallet"`) || !strings.Contains(m, `"version_number": "1.2.3"`) || !strings.Contains(m, `"dependencies"`) {
		t.Errorf("manifest %s", m)
	}
	if !strings.Contains(r.Log, "Packed 4 files") {
		t.Errorf("log %q", r.Log)
	}
}
