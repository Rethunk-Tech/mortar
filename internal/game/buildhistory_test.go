package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeManifest(t *testing.T, lib, manifest string) {
	t.Helper()
	steamapps := filepath.Join(lib, "steamapps")
	if err := os.MkdirAll(filepath.Join(steamapps, "common", "G"), 0o750); err != nil {
		t.Fatal(err)
	}
	acf := fmt.Sprintf("\"AppState\"\n{\n\"buildid\" \"b%s\"\n\"InstalledDepots\"\n{\n\"2\"\n{\n\"manifest\" \"%s\"\n\"size\" \"5\"\n}\n}\n}\n", manifest, manifest)
	if err := os.WriteFile(filepath.Join(steamapps, "appmanifest_1.acf"), []byte(acf), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestBuildHistory(t *testing.T) {
	data, lib := t.TempDir(), t.TempDir()
	dir := filepath.Join(lib, "steamapps", "common", "G")
	if got := BuildHistory(data, "g", dir); len(got) != 0 {
		t.Fatalf("a data folder without the file lists %v", got)
	}
	day := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	writeManifest(t, lib, "10")
	RecordBuildsIn(data, "g", "1", dir, day)
	RecordBuildsIn(data, "g", "1", dir, day.AddDate(0, 0, 1))
	if got := BuildHistory(data, "g", dir); len(got) != 1 || got[0].Manifest != "10" || got[0].BuildID != "b10" || got[0].Seen != "2026-01-01" || !got[0].Current {
		t.Fatalf("unchanged manifest: %+v", got)
	}
	writeManifest(t, lib, "11")
	RecordBuildsIn(data, "g", "1", dir, day.AddDate(0, 0, 2))
	got := BuildHistory(data, "g", dir)
	if len(got) != 2 || got[0].Manifest != "11" || !got[0].Current || got[1].Manifest != "10" || got[1].Current {
		t.Fatalf("new manifest: %+v", got)
	}
	for i := 12; i < 12+60; i++ {
		writeManifest(t, lib, fmt.Sprint(i))
		RecordBuildsIn(data, "g", "1", dir, day.AddDate(0, 0, i))
	}
	got = BuildHistory(data, "g", dir)
	if len(got) != maxBuildsPerDepot || got[0].Manifest != "71" || got[len(got)-1].Manifest != "22" {
		t.Fatalf("cap: len %d first %s last %s", len(got), got[0].Manifest, got[len(got)-1].Manifest)
	}
}
