package pack

import (
	"archive/zip"
	"bytes"
	"slices"
	"testing"
)

func TestZipKeepsFileOrder(t *testing.T) {
	t.Parallel()
	z, err := Zip(SortedFiles(map[string][]byte{"b": []byte("2"), "a": []byte("1"), "c": nil}))
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(z), int64(len(z)))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range zr.File {
		got = append(got, f.Name)
	}
	if !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("entries = %v", got)
	}
}
