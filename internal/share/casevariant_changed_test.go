package share

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteKeepsOneOfTwoChangedCopiesDifferingByCase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("two names differing only by case cannot sit side by side on a case-insensitive Windows folder")
	}
	dir, p := changedProfile(t)
	writeIn(t, dir, "changed/Mods/mc/A.cfg", []byte("1"))
	writeIn(t, dir, "changed/Mods/mc/a.cfg", []byte("2"))
	var buf bytes.Buffer
	skipped, err := Write(&buf, "sims4", p, filepath.Join(dir, "mods"), OwnInclude())
	if err != nil {
		t.Fatal(err)
	}
	pv, err := ReadBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("a written file must read back: %v (skipped %v)", err, skipped)
	}
	if len(skipped) != 1 {
		t.Fatalf("changed = %+v, skipped = %v", pv.ChangedFiles, skipped)
	}
}
