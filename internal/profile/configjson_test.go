package profile

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestRewriteConfigJSONKeepsOrderAndTypes(t *testing.T) {
	t.Parallel()
	in := []byte(`{"z":1,"a":{"n":1.5,"flag":true,"s":"x","xs":["p","q"],"extra":null},"mixed":[1,"a"]}`)
	got, err := rewriteConfigJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`{
  "z": 1,
  "a": {
    "n": 1.5,
    "flag": true,
    "s": "x",
    "xs": [
      "p",
      "q"
    ],
    "extra": null
  },
  "mixed": [
    1,
    "a"
  ]
}
`)
	if !bytes.Equal(got, want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	again, err := rewriteConfigJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, want) {
		t.Fatalf("second pass:\n%s", again)
	}
}

func TestConfigInModJoinsTheConfigOntoTheModFolder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	got, err := configInMod(root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != filepath.Clean(root) || filepath.Base(got) != configFile {
		t.Fatalf("config = %q", got)
	}
}
