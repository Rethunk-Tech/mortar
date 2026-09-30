package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestParseLenient(t *testing.T) {
	cases := map[string]struct {
		in   string
		want Manifest
	}{
		"bom comments trailing commas": {
			"\xef\xbb\xbf{\n  // the mod\n  \"Name\": \"Content Patcher\", /* inline */\n  \"Author\": \"Pathoschild\",\n  \"Version\": \"2.0.0\",\n  \"Description\": \"http://x // not a comment\",\n  \"UniqueID\": \"Pathoschild.ContentPatcher\",\n  \"Dependencies\": [ { \"UniqueID\": \"a\", }, ],\n}\n",
			Manifest{"Content Patcher", "Pathoschild", "2.0.0", "Pathoschild.ContentPatcher", "http://x // not a comment"},
		},
		"lowercase keys": {
			`{"name":"N","author":"A","version":"1.2","uniqueid":"A.N","description":"d"}`,
			Manifest{"N", "A", "1.2", "A.N", "d"},
		},
		"legacy version object": {
			`{"Name":"Old","Author":"A","UniqueID":"A.Old","Version":{"MajorVersion":1,"MinorVersion":2,"PatchVersion":3,"Build":"beta"}}`,
			Manifest{Name: "Old", Author: "A", Version: "1.2.3-beta", UniqueID: "A.Old"},
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
