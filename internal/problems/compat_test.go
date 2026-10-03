package problems

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/meta"
)

func TestMatchCompatByUniqueIDAndNexus(t *testing.T) {
	idx := meta.CompatIndex{
		ByID: map[string]meta.CompatEntry{
			"author.broken": {Status: meta.StatusBroken, Summary: "Crashes.", BrokeIn: "1.6"},
			"author.ok":     {Status: meta.StatusOK},
		},
		ByNexus: map[int]meta.CompatEntry{
			42: {Status: meta.StatusUnofficial, UnofficialURL: "https://example.com/u", Summary: "Patch."},
		},
	}
	mods := []Installed{
		{Key: "a", Manifest: manifest.Manifest{UniqueID: "Author.Broken", Name: "Broken"}},
		{Key: "b", Manifest: manifest.Manifest{UniqueID: "Author.NexusOnly", Name: "Nexus", UpdateKeys: []string{"Nexus:42"}}},
		{Key: "c", Manifest: manifest.Manifest{UniqueID: "Author.Ok", Name: "Fine"}},
		{Key: "d", Manifest: manifest.Manifest{UniqueID: "Author.Unknown", Name: "Unknown"}},
	}
	all := matchCompat(idx, mods, false)
	if len(all) != 3 {
		t.Fatalf("all: %+v", all)
	}
	info := matchCompat(idx, mods, true)
	if len(info) != 2 {
		t.Fatalf("non-ok: %+v", info)
	}
	if info[0].Status != meta.StatusBroken || info[0].UniqueID != "Author.Broken" {
		t.Fatalf("id map: %+v", info[0])
	}
	if info[1].Status != meta.StatusUnofficial || info[1].UnofficialURL != "https://example.com/u" {
		t.Fatalf("nexus map: %+v", info[1])
	}
}
