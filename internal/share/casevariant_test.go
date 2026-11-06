package share

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestWriteKeepsOneOfTwoConfigNamesDifferingByCase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("two names differing only by case cannot sit side by side on a case-insensitive Windows folder")
	}
	dir := t.TempDir()
	writeIn(t, dir, "mods/k/Config.json", []byte(`{"a":1}`))
	writeIn(t, dir, "mods/k/config.json", []byte(`{"a":2}`))
	p := profile.Profile{Name: "P", Entries: []profile.Entry{{
		Key: "k", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 70},
		Mods: []profile.Component{{ID: "smapi:A.Mod", Folder: "."}},
	}}}
	pv, skipped := writeAndRead(t, "stardew", p, dir)
	if len(pv.Configs) != 1 || pv.Configs[0].Path != "Config.json" || len(skipped) != 1 {
		t.Fatalf("configs = %+v, skipped = %v", pv.Configs, skipped)
	}
}

// writeAndRead writes p's share from dir's mods folder and reads it back, returning the preview and what the write skipped.
func writeAndRead(t *testing.T, game string, p profile.Profile, dir string, include ...Include) (Preview, []string) {
	t.Helper()
	var buf bytes.Buffer
	skipped, err := Write(&buf, game, p, filepath.Join(dir, "mods"), include...)
	if err != nil {
		t.Fatal(err)
	}
	pv, err := ReadBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("a written file must read back: %v (skipped %v)", err, skipped)
	}
	return pv, skipped
}
