package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestParseLenient(t *testing.T) {
	cases := map[string]struct {
		in   string
		want Manifest
	}{
		"bom comments trailing commas": {
			"\xef\xbb\xbf{\n  // the mod\n  \"Name\": \"Content Patcher\", /* inline */\n  \"Author\": \"Pathoschild\",\n  \"Version\": \"2.0.0\",\n  \"Description\": \"http://x // not a comment\",\n  \"UniqueID\": \"Pathoschild.ContentPatcher\",\n  \"Dependencies\": [ { \"UniqueID\": \"a\", }, ],\n}\n",
			Manifest{Name: "Content Patcher", Author: "Pathoschild", Version: "2.0.0", UniqueID: "Pathoschild.ContentPatcher", Description: "http://x // not a comment", Dependencies: []Dependency{{UniqueID: "a", Required: true}}},
		},
		"lowercase keys": {
			`{"name":"N","author":"A","version":"1.2","uniqueid":"A.N","description":"d"}`,
			Manifest{Name: "N", Author: "A", Version: "1.2", UniqueID: "A.N", Description: "d"},
		},
		"dependencies and content pack": {
			`{"UniqueID":"A.P","UpdateKeys":["Nexus:1"," ",7],"Dependencies":[{"UniqueID":"B.Req","MinimumVersion":"1.2-beta"},{"uniqueid":"B.Opt","isrequired":false},{"MinimumVersion":"1"},"junk"],"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"2.0"}}`,
			Manifest{
				UniqueID: "A.P", UpdateKeys: []string{"Nexus:1"}, ContentPackFor: "Pathoschild.ContentPatcher",
				Dependencies: []Dependency{
					{"B.Req", "1.2-beta", true, ""}, {"B.Opt", "", false, ""}, {"Pathoschild.ContentPatcher", "2.0", true, ""},
				},
			},
		},
		"framework listed twice": {
			`{"UniqueID":"A.P","Dependencies":[{"UniqueID":"pathoschild.contentpatcher","IsRequired":false}],"ContentPackFor":{"UniqueID":"Pathoschild.ContentPatcher","MinimumVersion":"2.0"}}`,
			Manifest{
				UniqueID: "A.P", ContentPackFor: "Pathoschild.ContentPatcher",
				Dependencies: []Dependency{{"pathoschild.contentpatcher", "2.0", true, ""}},
			},
		},
		"legacy version object": {
			`{"Name":"Old","Author":"A","UniqueID":"A.Old","Version":{"MajorVersion":1,"MinorVersion":2,"PatchVersion":3,"Build":"beta"}}`,
			Manifest{Name: "Old", Author: "A", Version: "1.2.3-beta", UniqueID: "A.Old"},
		},
		"update hints case insensitive": {
			`{"UniqueID":"A.B","UpdateCautionMessage":"Read the release notes","DELETEOLDVERSION":true}`,
			Manifest{UniqueID: "A.B", UpdateCautionMessage: "Read the release notes", DeleteOldVersion: true},
		},
		"escaped quote in string": {
			`{"Name":"Say \"hi\" // ok","UniqueID":"A.B","Version":"1.0"}`,
			Manifest{Name: `Say "hi" // ok`, Version: "1.0", UniqueID: "A.B"},
		},
	}
	for name, c := range cases {
		got, err := Parse([]byte(c.in))
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %+v, %v; want %+v", name, got, err, c.want)
		}
	}
	for _, bad := range []string{`{"Name":"x"}`, `not json`, `{"UniqueID":"a"`} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
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

func TestParseReadsStringIsRequired(t *testing.T) {
	m, err := Parse([]byte(`{"UniqueID": "A", "Dependencies": [
		{"UniqueID": "Opt", "IsRequired": "false"},
		{"UniqueID": "Req", "IsRequired": "True"},
		{"UniqueID": "Plain"}
	]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Opt": false, "Req": true, "Plain": true}
	for _, d := range m.Dependencies {
		if w, ok := want[d.UniqueID]; ok && d.Required != w {
			t.Errorf("%s Required = %v, want %v", d.UniqueID, d.Required, w)
		}
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

func TestGitHubRepoRefusesWhatMovesTheURL(t *testing.T) {
	for _, repo := range []string{"me/..", "./mod", "../..", "me/.", "o/r?x", "o/r#x", "o/r/s", "o /r"} {
		if ValidGitHubRepo(repo) {
			t.Errorf("ValidGitHubRepo(%q) = true", repo)
		}
		if _, ok := GitHubUpdateKey("GitHub:" + repo); ok {
			t.Errorf("GitHubUpdateKey accepted %q", repo)
		}
	}
	if !ValidGitHubRepo("me/mod.cfg") {
		t.Error("a dotted repo name was refused")
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

func TestParseNewtonsoftForms(t *testing.T) {
	utf16le := []byte{0xff, 0xfe}
	for _, c := range []byte(`{'UniqueID': 'A.U16'}`) {
		utf16le = append(utf16le, c, 0)
	}
	for name, tc := range map[string]struct {
		in   []byte
		want string
	}{
		"single quotes":         {[]byte(`{'Name':'N','UniqueID':'A.Single'}`), "A.Single"},
		"unquoted names":        {[]byte(`{Name:"N", UniqueID: "A.Bare",}`), "A.Bare"},
		"utf16":                 {utf16le, "A.U16"},
		"raw newline in string": {[]byte("{\"Description\":\"two\nlines\",\"UniqueID\":\"A.NL\"}"), "A.NL"},
		"curly quotes":          {[]byte("{“UniqueID”: “A.Curly”}"), "A.Curly"},
	} {
		m, err := Parse(tc.in)
		if err != nil || m.UniqueID != tc.want {
			t.Errorf("%s: %+v, %v", name, m, err)
		}
	}
}

func TestParseErrorsSayWhy(t *testing.T) {
	for in, want := range map[string]string{
		"{\n\"UniqueID\": \"A\",\n\"Name\" \"x\"\n}": "invalid JSON at line 3",
		`{"Name":"x"}`: "manifest has no UniqueID",
		`[1]`:          "not a JSON object",
		`null`:         "not a JSON object",
	} {
		if _, err := Parse([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", in, err, want)
		}
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
