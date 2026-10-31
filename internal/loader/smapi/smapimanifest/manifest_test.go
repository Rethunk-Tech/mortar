package smapimanifest

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// Parse carries every field smapi.Parse reads into Mortar's Manifest, IsRequired becoming Required.
func TestParseMapsEveryField(t *testing.T) {
	got, err := Parse([]byte(`{"Name":"N","Author":"A","Version":"1.2","UniqueID":"A.P","Description":"d","EntryDll":"P.dll",
		"UpdateKeys":["Nexus:1"],"UpdateCautionMessage":"careful","MinimumGameVersion":"1.6","DeleteOldVersion":true,
		"Dependencies":[{"UniqueID":"B.Req","MinimumVersion":"1.2-beta"},{"UniqueID":"B.Opt","IsRequired":false}],
		"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher"}}`))
	want := Manifest{
		Name: "N", Author: "A", Version: "1.2", UniqueID: "A.P", Description: "d", EntryDll: "P.dll", UpdateKeys: []string{"Nexus:1"},
		UpdateCautionMessage: "careful", MinimumGameVersion: "1.6", DeleteOldVersion: true, ContentPackFor: "Pathoschild.ContentPatcher",
		Dependencies: []mod.Dependency{
			{UniqueID: "B.Req", MinimumVersion: "1.2-beta", Required: true}, {UniqueID: "B.Opt"}, {UniqueID: "Pathoschild.ContentPatcher", Required: true},
		},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// Order hands SMAPI's resolver each mod's dependencies and maps its rows back to SMAPI mod ids.
func TestOrderMapsRowsBack(t *testing.T) {
	rows := Order([]Mod{
		{UniqueID: "Me.Pack", Name: "Pack", ContentPackFor: "Me.Frame", Dependencies: []mod.Dependency{{UniqueID: "Me.Opt"}, {UniqueID: "Me.Gone", Required: true}}, Folder: "a"},
		{UniqueID: "Me.Frame", Name: "Frame", Folder: "b"},
		{UniqueID: "Me.Opt", Name: "Opt", Folder: "c"},
	})
	if len(rows) != 3 || rows[1].ID != "smapi:Me.Frame" || rows[1].Position != 2 || !reflect.DeepEqual(rows[1].Dependents, []mod.ID{"smapi:Me.Pack"}) {
		t.Fatalf("rows = %+v", rows)
	}
	if pack := rows[0]; pack.ID != "smapi:Me.Pack" || !reflect.DeepEqual(pack.Optional, []mod.ID{"smapi:Me.Opt"}) ||
		!reflect.DeepEqual(pack.MissingRequired, []mod.ID{"smapi:Me.Gone"}) {
		t.Fatalf("pack = %+v", pack)
	}
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	testfs.WriteAt(t, p, []byte(body))
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	write(t, root, "Pack/manifest.json", `{"UniqueID":"A.Pack"}`)
	write(t, root, "Pack/assets/deep/manifest.json", `{"UniqueID":"A.Inside"}`)
	write(t, root, "Wrapper/Inner/manifest.json", `{"UniqueID":"A.Inner"}`)
	write(t, root, "Wrapper/Other/manifest.json", `{"UniqueID":"A.Other"}`)
	write(t, root, ".off/manifest.json", `{"UniqueID":"A.Off"}`)
	write(t, root, "Wrapper/.hidden/manifest.json", `{"UniqueID":"A.Hidden"}`)
	write(t, root, "Broken/manifest.json", `{"Name":"no id"}`)
	got, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range got {
		ids = append(ids, m.Folder+"="+m.UniqueID)
	}
	want := []string{"Pack=A.Pack", "Wrapper/Inner=A.Inner", "Wrapper/Other=A.Other"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("scan = %v, want %v", ids, want)
	}

	rootMod := t.TempDir()
	write(t, rootMod, "manifest.json", `{"UniqueID":"A.Root"}`)
	write(t, rootMod, "sub/manifest.json", `{"UniqueID":"A.Sub"}`)
	got, err = Scan(rootMod)
	if err != nil || len(got) != 1 || got[0].Folder != "." {
		t.Fatalf("root scan = %+v, %v", got, err)
	}
}

func TestScanSkipsASymlinkDirectory(t *testing.T) {
	root := t.TempDir()
	write(t, root, "Pack/manifest.json", `{"UniqueID":"A.Pack"}`)
	outside := t.TempDir()
	write(t, outside, "manifest.json", `{"UniqueID":"A.Outside"}`)
	if err := os.Symlink(outside, filepath.Join(root, "Innocent")); err != nil {
		t.Fatal(err)
	}
	got, err := Scan(root)
	if err != nil || len(got) != 1 || got[0].UniqueID != "A.Pack" {
		t.Fatalf("scan = %+v, %v", got, err)
	}
}

func TestModIDIsSMAPIScoped(t *testing.T) {
	m := Manifest{UniqueID: "Au.One", ContentPackFor: "Au.Two"}
	if m.ModID() != "smapi:Au.One" || m.ContentPackForID() != "smapi:Au.Two" || (Manifest{}).ContentPackForID() != "" {
		t.Errorf("ids = %q, %q", m.ModID(), m.ContentPackForID())
	}
	if !LoaderManaged("bepinex:Rethunk.MortarBepInExBridge") || LoaderManaged("bepinex:Au.One") {
		t.Error("LoaderManaged must hide the BepInEx bridge")
	}
	if !LoaderManaged("smapi:SMAPI.ConsoleCommands") || LoaderManaged("smapi:Au.One") {
		t.Error("LoaderManaged must match SMAPI's bundled mods by folded id")
	}
}

// A manifest parsed again comes from the cache; a caller changing its copy does not change the next one.
func TestParseAgainReturnsAnUnsharedCopy(t *testing.T) {
	raw := []byte(`{"UniqueID":"Me.Cached","UpdateKeys":["Nexus:1"],"Dependencies":[{"UniqueID":"Me.Dep"}]}`)
	first, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	first.UpdateKeys[0], first.Dependencies[0].UniqueID = "changed", "changed"
	again, err := Parse(raw)
	if err != nil || again.UpdateKeys[0] != "Nexus:1" || again.Dependencies[0].UniqueID != "Me.Dep" {
		t.Fatalf("second parse = %+v %v", again, err)
	}
	if _, err := Parse([]byte(`{"Name":"no id"}`)); err == nil {
		t.Fatal("a manifest without a UniqueID parsed")
	}
}

// Vortex deploys by linking each mod file from its staging folder into the game's Mods folder.
func TestScanFollowsDeployedManifestLinks(t *testing.T) {
	staging, mods := t.TempDir(), t.TempDir()
	for _, m := range []struct{ folder, id string }{{"Top", "A.Top"}, {"Pack/Nested", "A.Nested"}} {
		src := filepath.Join(staging, m.folder)
		if err := os.MkdirAll(src, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(src, FileName), []byte(`{"UniqueID":"`+m.id+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(mods, m.folder)
		if err := os.MkdirAll(dst, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(src, FileName), filepath.Join(dst, FileName)); err != nil {
			t.Skip("symlinks unavailable:", err)
		}
	}
	if err := os.MkdirAll(filepath.Join(mods, "Dangling"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(staging, "gone.json"), filepath.Join(mods, "Dangling", FileName)); err != nil {
		t.Fatal(err)
	}
	found, err := Scan(mods)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, m := range found {
		got[m.Folder] = m.UniqueID
	}
	want := map[string]string{"Top": "A.Top", "Pack/Nested": "A.Nested"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Scan = %v, want %v", got, want)
	}
}

func TestScanInvalidReportsANestedFailure(t *testing.T) {
	root := t.TempDir()
	write(t, root, "Good/manifest.json", `{"UniqueID":"A.Good"}`)
	write(t, root, "Wrapper/Inner/manifest.json", `{"UniqueID": }`)
	mods, invalid, err := ScanInvalid(root)
	if err != nil || len(mods) != 1 || invalid == nil || invalid.Folder != "Wrapper/Inner" ||
		!strings.HasPrefix(invalid.Error(), "Wrapper/Inner/manifest.json: invalid JSON at line 1") {
		t.Fatalf("mods %v, invalid %+v, err %v", mods, invalid, err)
	}
}
