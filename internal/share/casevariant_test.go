package share

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestWriteKeepsOneOfTwoConfigNamesDifferingByCase(t *testing.T) {
	dir := t.TempDir()
	writeIn(t, dir, "mods/k/Config.json", []byte(`{"a":1}`))
	writeIn(t, dir, "mods/k/config.json", []byte(`{"a":2}`))
	p := profile.Profile{Name: "P", Entries: []profile.Entry{{
		Key: "k", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 70},
		Mods: []profile.Component{{ID: "smapi:A.Mod", Folder: "."}},
	}}}
	var buf bytes.Buffer
	skipped, err := Write(&buf, "stardew", p, filepath.Join(dir, "mods"))
	if err != nil {
		t.Fatal(err)
	}
	pv, err := ReadBytes(buf.Bytes())
	if err != nil {
		t.Fatalf("a written file must read back: %v (skipped %v)", err, skipped)
	}
	if len(pv.Configs) != 1 || pv.Configs[0].Path != "Config.json" || len(skipped) != 1 {
		t.Fatalf("configs = %+v, skipped = %v", pv.Configs, skipped)
	}
}
