package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

func TestOpenDropsInvalidLastPlayed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	raw := Settings{
		Accent:     "sand",
		Background: BackgroundImage,
		LastPlayed: map[string]Played{
			"stardew": {Profile: "p1", At: "not-a-time"},
			"":        {Profile: "p2", At: time.Now().UTC().Format(time.RFC3339)},
			"lethal":  {Profile: "", At: time.Now().UTC().Format(time.RFC3339)},
			"ok":      {Profile: "keep", At: "2026-09-30T16:00:00Z"},
		},
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), b, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	got := s.Get().LastPlayed
	if len(got) != 1 || got["ok"].Profile != "keep" || got["ok"].At != "2026-09-30T16:00:00Z" {
		t.Fatalf("LastPlayed = %#v", got)
	}
}

func TestRecordLastPlayed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	s, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC)
	if _, err := s.RecordLastPlayed("stardew", "prof-1", at, "1.6.15"); err != nil {
		t.Fatal(err)
	}
	got := s.Get().LastPlayed["stardew"]
	if got.Profile != "prof-1" || got.At != "2026-09-30T16:00:00Z" || got.GameVersion != "1.6.15" {
		t.Fatalf("got %#v", got)
	}
	if _, err := s.RecordLastPlayed("stardew", "prof-1", at, ""); err != nil {
		t.Fatal(err)
	}
	if s.Get().LastPlayed["stardew"].GameVersion != "1.6.15" {
		t.Fatal("empty version must keep the last recorded game version")
	}
}
