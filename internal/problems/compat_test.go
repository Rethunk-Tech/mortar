package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
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
	mods := []framework.Mod{
		{Key: "a", UniqueID: "Author.Broken", Name: "Broken"},
		{Key: "b", UniqueID: "Author.NexusOnly", Name: "Nexus", UpdateKeys: []string{"Nexus:42"}},
		{Key: "c", UniqueID: "Author.Ok", Name: "Fine"},
		{Key: "d", UniqueID: "Author.Unknown", Name: "Unknown"},
	}
	all := matchCompat(idx, mods, false)
	if len(all) != 3 {
		t.Fatalf("all: %+v", all)
	}
	info := matchCompat(idx, mods, true)
	if len(info) != 2 {
		t.Fatalf("non-ok: %+v", info)
	}
	if info[0].Status != meta.StatusBroken || info[0].ID != "smapi:Author.Broken" {
		t.Fatalf("id map: %+v", info[0])
	}
	if info[1].Status != meta.StatusUnofficial || info[1].UnofficialURL != "https://example.com/u" {
		t.Fatalf("nexus map: %+v", info[1])
	}
}

func TestMatchCompatSkipsUnofficialRowOnceInstalledMeetsIt(t *testing.T) {
	idx := meta.CompatIndex{ByID: map[string]meta.CompatEntry{
		"bungus.showitemquality": {Status: meta.StatusUnofficial, UnofficialVersion: "1.1.3-unofficial.1-bungus"},
		"hootless.buslocations":  {Status: meta.StatusUnofficial, UnofficialVersion: "1.2.2-unofficial.1-Xytronix"},
		"author.old":             {Status: meta.StatusUnofficial, UnofficialVersion: "2.0.0-unofficial.1-x"},
	}}
	mods := []framework.Mod{
		{Key: "a", UniqueID: "bungus.ShowItemQuality", Version: "1.1.3-unofficial.1-bungus"},
		{Key: "b", UniqueID: "hootless.BusLocations", Version: "2.1.1"},
		{Key: "c", UniqueID: "author.Old", Version: "1.0.0"},
		{Key: "d", UniqueID: "author.Old", Version: "not a version"},
	}

	got := matchCompat(idx, mods, true)

	if len(got) != 2 || got[0].Key != "c" || got[1].Key != "d" {
		t.Fatalf("matchCompat() = %+v, want only the older and the unparseable install", got)
	}
}
