package profile

import (
	"bytes"
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

func TestConfigInModRejectsPathsOutsideTheFolder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := configInMod(root, configFile); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"../config.json", "/tmp/config.json", "nested/config.json", "config.json/../x"} {
		if _, err := configInMod(root, rel); err == nil {
			t.Errorf("%s: accepted", rel)
		}
	}
}
