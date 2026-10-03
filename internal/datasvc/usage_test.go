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

func TestMeasureModUsageAggregatesStoreAndProfileCopies(t *testing.T) {
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
	write("store/stardew/nexus-1-1/m.bin", 100)
	write("store/stardew/local-aa/m.bin", 40)
	write("profiles/stardew/aaa/profile.json", 1)
	write("profiles/stardew/aaa/mods/nexus-1-1/copy.bin", 80)
	write("profiles/stardew/bbb/profile.json", 1)
	write("profiles/stardew/bbb/mods/nexus-1-1/copy.bin", 90)
	if err := os.WriteFile(filepath.Join(root, "profiles", "stardew", "aaa", "profile.json"), []byte(
		`{"id":"aaa","name":"A","entries":[{"key":"nexus-1-1","source":{"name":"Alpha","version":"1.0"}}]}`,
	), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profiles", "stardew", "bbb", "profile.json"), []byte(
		`{"id":"bbb","name":"B","entries":[{"key":"nexus-1-1","source":{"name":"Alpha","version":"1.0"}}]}`,
	), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "store", "index.json"), []byte(
		`{"stardew":{"nexus-1-1":"2026-01-02T03:04:05Z"}}`,
	), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := MeasureMods(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 140 {
		t.Fatalf("total = %d", got.Total)
	}
	byKey := map[string]ModUse{}
	for _, it := range got.Items {
		byKey[it.Key] = it
	}
	a := byKey["nexus-1-1"]
	if a.Size != 100 || a.Profiles != 2 || a.ProfileSize != 170 || a.Name != "Alpha 1.0" || a.LastUsed != "2026-01-02T03:04:05Z" {
		t.Fatalf("nexus-1-1 = %+v", a)
	}
	b := byKey["local-aa"]
	if b.Size != 40 || b.Profiles != 0 || b.ProfileSize != 0 {
		t.Fatalf("local-aa = %+v", b)
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
