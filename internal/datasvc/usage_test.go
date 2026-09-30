package datasvc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureSizesSkipSymlinks(t *testing.T) {
	root := t.TempDir()
	write := func(rel string, n int) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, make([]byte, n), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("profiles/stardew/0123456789abcdef/profile.json", 20)
	write("profiles/stardew/0123456789abcdef/mods/a.bin", 100)
	write("store/stardew/smapi-1/m.bin", 50)
	write("cache/nexus/x.json", 7)
	write("backups/one.zip", 11)
	write("trash/stardew/dead/mods/b.bin", 13)
	write("settings.json", 3)
	target := filepath.Join(root, "store", "stardew", "smapi-1", "m.bin")
	link := filepath.Join(root, "cache", "link.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	got, err := Measure(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Store != 50 || got.Cache != 7 || got.Backups != 11 || got.Trash != 13 {
		t.Fatalf("buckets store=%d cache=%d backups=%d trash=%d", got.Store, got.Cache, got.Backups, got.Trash)
	}
	if got.Total != 20+100+50+7+11+13+3 {
		t.Fatalf("total = %d", got.Total)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Size != 100 || got.Profiles[0].ID != "0123456789abcdef" {
		t.Fatalf("profiles = %+v", got.Profiles)
	}
}

func TestMeasureReportsProgress(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "store", "f.bin")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	var last Progress
	if _, err := Measure(root, func(p Progress) { last = p }); err != nil {
		t.Fatal(err)
	}
	if !last.Measuring || last.Bytes != 3 {
		t.Fatalf("progress = %+v", last)
	}
}
