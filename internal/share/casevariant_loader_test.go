package share

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestWriteKeepsOneOfTwoLoaderConfigsDifferingByCase(t *testing.T) {
	dir := t.TempDir()
	writeIn(t, dir, "BepInEx/config/A.cfg", []byte("1"))
	writeIn(t, dir, "BepInEx/config/a.cfg", []byte("2"))
	p := profile.Profile{Name: "P", Loader: "bepinex5"}
	var buf bytes.Buffer
	skipped, err := Write(&buf, "lethal-company", p, filepath.Join(dir, "mods"), OwnInclude())
	if err != nil {
		t.Fatal(err)
	}
	pv, err := ReadBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("a written file must read back: %v (skipped %v)", err, skipped)
	}
	if len(pv.LoaderConfigs) != 1 || len(skipped) != 1 {
		t.Fatalf("loader configs = %+v, skipped = %v", pv.LoaderConfigs, skipped)
	}
}
