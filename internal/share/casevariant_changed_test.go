package share

import (
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
	pv, skipped := writeAndRead(t, "sims4", p, dir, OwnInclude())
	if len(skipped) != 1 {
		t.Fatalf("changed = %+v, skipped = %v", pv.ChangedFiles, skipped)
	}
}
