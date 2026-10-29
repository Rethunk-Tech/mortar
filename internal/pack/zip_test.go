package pack

import (
	"archive/zip"
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/archive"
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

// A pack's files land in a game folder, where two spellings of one name collide on Windows and macOS, so a pack
// holding both is refused rather than keeping either.
func TestReadZipRefusesCaseVariants(t *testing.T) {
	t.Parallel()
	z, err := Zip(SortedFiles(map[string][]byte{"Mods/a.dll": []byte("1"), "mods/b.dll": []byte("2")}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readZip(z); !errors.Is(err, archive.ErrCaseCollision) {
		t.Fatalf("err = %v", err)
	}
}
