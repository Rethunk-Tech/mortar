package datasvc

import (
	"github.com/Rethunk-Tech/mortar/internal/store"
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
	if got.Store < 50 || got.Cache < 7 || got.Backups < 11 || got.Trash < 13 {
		t.Fatalf("buckets store=%d cache=%d backups=%d trash=%d", got.Store, got.Cache, got.Backups, got.Trash)
	}
	if got.Total < 20+100+50+7+11+13+3 {
		t.Fatalf("total = %d", got.Total)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Size < 100 || got.Profiles[0].ID != "0123456789abcdef" {
		t.Fatalf("profiles = %+v", got.Profiles)
	}
}

func TestMeasureGameTotals(t *testing.T) {
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
	write("profiles/alpha/p1/profile.json", 4)
	write("profiles/alpha/p1/mods/a.bin", 100)
	write("store/alpha/k/m.bin", 20)
	write("store/.manifests/alpha/k.json", 2)
	write("backups/alpha/s.zip", 8)
	write("cache/alpha/c.bin", 5)
	write("cache/nexus/alpha/n.json", 3)
	write("profiles/beta/p1/profile.json", 2)
	write("profiles/beta/p1/mods/b.bin", 50)
	write("store/beta/k/m.bin", 10)
	write("backups/beta/s.zip", 7)
	write("cache/beta/c.bin", 6)
	write("cache/nexus/beta/n.json", 1)
	write("cache/nexus/x.json", 9)
	write("backups/one.zip", 11)
	write("trash/alpha/dead.bin", 40)
	write("profiles/stardew/p1/profile.json", 1)
	got, err := Measure(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	byGame := map[string]int64{}
	for _, g := range got.Games {
		byGame[g.Game] = g.Size
		want := g.Game
		if g.Game == "stardew" {
			want = "Stardew Valley"
		}
		if g.Name != want {
			t.Fatalf("name = %q, want %q", g.Name, want)
		}
	}
	if byGame["alpha"] < 4+100+20+8+5+3 {
		t.Fatalf("alpha = %d games=%+v", byGame["alpha"], got.Games)
	}
	if byGame["beta"] < 2+50+10+7+6+1 {
		t.Fatalf("beta = %d games=%+v", byGame["beta"], got.Games)
	}
	if _, ok := byGame["nexus"]; ok {
		t.Fatalf("cache kind counted as a game: %+v", got.Games)
	}
	if _, ok := byGame[".manifests"]; ok {
		t.Fatalf("store index listed as a game: %+v", got.Games)
	}
	if _, ok := byGame["one.zip"]; ok {
		t.Fatalf("loose backup counted as a game: %+v", got.Games)
	}
	if len(byGame) != 3 {
		t.Fatalf("games = %+v", got.Games)
	}
}

func TestMeasureSharedSavedHardlink(t *testing.T) {
	root := t.TempDir()
	store := filepath.Join(root, "store", "g", "k", "m.bin")
	if err := os.MkdirAll(filepath.Dir(store), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store, make([]byte, 8192), 0o600); err != nil {
		t.Fatal(err)
	}
	prof := filepath.Join(root, "profiles", "g", "0123456789abcdef", "mods", "k", "m.bin")
	if err := os.MkdirAll(filepath.Dir(prof), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(store, prof); err != nil {
		t.Fatal(err)
	}
	got, err := Measure(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !got.SharedSavedKnown || got.SharedSaved < 4000 {
		t.Fatalf("sharedSaved known=%v n=%d", got.SharedSavedKnown, got.SharedSaved)
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
	items := store.OpenAt(filepath.Join(root, "store"))
	for key, n := range map[string]int{"nexus-1-1": 100, "local-aa": 40} {
		src := t.TempDir()
		if err := os.WriteFile(filepath.Join(src, "m.bin"), make([]byte, n), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := items.AddDir("stardew", key, src); err != nil {
			t.Fatal(err)
		}
	}
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
	if a.Size != 100 || a.Profiles != 2 || a.ProfileSize != 170 || a.Name != "Alpha 1.0" || a.LastUsed == "" {
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
	if !last.Measuring || last.Bytes < 3 {
		t.Fatalf("progress = %+v", last)
	}
}
