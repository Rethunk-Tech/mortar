package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

func TestNewestChannelFile(t *testing.T) {
	t.Parallel()
	files := []nexus.File{
		{FileID: 1, Category: "MAIN", Version: "1.0.0", Name: "Mod"},
		{FileID: 2, Category: "MAIN", Version: "1.1.0", Name: "Mod"},
		{FileID: 3, Category: "OPTIONAL", Version: "1.2.0", Name: "Mod extra"},
		{FileID: 4, Category: "MAIN", Version: "1.3.0-beta", Name: "Mod"},
		{FileID: 5, Category: "OPTIONAL", Version: "1.4.0", Name: "Mod alpha pack", FileName: "mod-alpha.zip"},
		{FileID: 6, Category: "OLD_VERSION", Version: "2.0.0", Name: "Mod old"},
	}
	tests := []struct {
		channel string
		want    int
	}{
		{channel: "", want: 2},
		{channel: "main", want: 2},
		{channel: "optional", want: 3},
		{channel: "beta", want: 5},
	}
	for _, tc := range tests {
		got, ok := NewestChannelFile(tc.channel, "1.0.0", files)
		if !ok || got.FileID != tc.want {
			t.Fatalf("channel %q: got %+v ok=%v, want file %d", tc.channel, got, ok, tc.want)
		}
	}
}

func TestSetUpdateChannelOneHistoryEvent(t *testing.T) {
	t.Parallel()
	s := newStore(t)
	p := pinTestProfile(t, s)
	before, err := s.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.SetUpdateChannel("stardew", p.ID, "nexus-1-1", "optional")
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].UpdateChannel != ChannelOptional {
		t.Fatalf("channel: %q", p.Entries[0].UpdateChannel)
	}
	after, err := s.History("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("history %d -> %d", len(before), len(after))
	}
	if after[0].Kind != historyChannel || after[0].Name != entryName(p.Entries[0]) || after[0].Detail != ChannelOptional {
		t.Fatalf("event: %+v, want the mod %q switched to %q", after[0], entryName(p.Entries[0]), ChannelOptional)
	}
	p, err = s.SetUpdateChannel("stardew", p.ID, "nexus-1-1", "main")
	if err != nil {
		t.Fatal(err)
	}
	if p.Entries[0].UpdateChannel != "" {
		t.Fatalf("main should store empty, got %q", p.Entries[0].UpdateChannel)
	}
	if _, err := s.SetUpdateChannel("stardew", p.ID, "nexus-1-1", "nightly"); err == nil {
		t.Fatal("nightly must fail")
	}
}
