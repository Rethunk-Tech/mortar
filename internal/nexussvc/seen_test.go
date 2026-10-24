package nexussvc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkSeenKeepsGamesAndSourcesApart(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	s := &Service{}
	if err := s.MarkNexusSeen("stardew", 541, 1, ""); err == nil {
		t.Fatal("marking before the store opens must fail")
	}
	if err := s.UseDataDir(dir); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"", "541"} {
		if err := s.MarkSeen(k, 1, "1"); err == nil {
			t.Fatalf("key %q accepted", k)
		}
	}
	if err := s.MarkNexusSeen("stardew", 541, 7, "1.2"); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkNexusSeen("lethal-company", 541, 9, "3.0"); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkSeen("github:me/mod", 0, "v3"); err != nil {
		t.Fatal(err)
	}
	s2 := &Service{}
	if err := s2.UseDataDir(dir); err != nil {
		t.Fatal(err)
	}
	got := s2.Seen()
	if got[NexusSeenKey("stardew", 541)].NewestChange != "1.2" || got[NexusSeenKey("lethal-company", 541)].NewestChange != "3.0" ||
		got["github:me/mod"].NewestChange != "v3" {
		t.Fatalf("reloaded %+v", got)
	}
}

func TestSeenStillReadsTheBareIDsEarlierReleasesWrote(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nexus-seen.json"), []byte(`{"mods":{"541":{"newestFileUnix":7,"newestChange":"1.2"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Service{}
	if err := s.UseDataDir(dir); err != nil {
		t.Fatal(err)
	}
	if got := s.Seen()["541"]; got.NewestChange != "1.2" || got.NewestFileUnix != 7 {
		t.Fatalf("%+v", got)
	}
}
