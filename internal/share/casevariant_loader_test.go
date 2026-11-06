package share

import (
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestWriteKeepsOneOfTwoLoaderConfigsDifferingByCase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("two names differing only by case cannot sit side by side on a case-insensitive Windows folder")
	}
	dir := t.TempDir()
	writeIn(t, dir, "BepInEx/config/A.cfg", []byte("1"))
	writeIn(t, dir, "BepInEx/config/a.cfg", []byte("2"))
	p := profile.Profile{Name: "P", Loader: "bepinex5"}
	pv, skipped := writeAndRead(t, "lethal-company", p, dir, OwnInclude())
	if len(pv.LoaderConfigs) != 1 || len(skipped) != 1 {
		t.Fatalf("loader configs = %+v, skipped = %v", pv.LoaderConfigs, skipped)
	}
}
