package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
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
	doc, err := encodeFile(raw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(doc)
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

func TestAddPlaytimeSurvivesRelaunch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	s, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPlaytime("stardew", time.Minute); err != nil {
		t.Fatal(err)
	}
	if len(s.Get().LastPlayed) != 0 {
		t.Fatal("playtime for a never-launched game must not create an entry")
	}
	at := time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC)
	if _, err := s.RecordLastPlayed("stardew", "p1", at, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPlaytime("stardew", 90*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RecordLastPlayed("stardew", "p2", at, ""); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().LastPlayed["stardew"].PlaytimeMs; got != 90_000 {
		t.Fatalf("PlaytimeMs = %d", got)
	}
}

func TestGameListColumnsStayWithTheirGame(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	s, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{store: s}
	if err := svc.SetListColumns("lethal-company", []string{"name", "order", "nope"}); err != nil {
		t.Fatal(err)
	}
	got := s.Get().GamePrefs("lethal-company").ListColumns
	if len(got) != 3 || got[0] != "on" || got[1] != "name" || got[2] != "order" {
		t.Fatalf("lethal-company columns = %v", got)
	}
	if len(s.Get().GamePrefs("stardew").ListColumns) != 0 {
		t.Fatal("another game must keep following the global list")
	}
}
