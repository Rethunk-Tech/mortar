package problems

import (
	"testing"

	"github.com/Rethunk-Tech/go-smapiweb"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestMatchCompatByUniqueIDAndNexus(t *testing.T) {
	idx := smapiweb.CompatIndex{
		ByID: map[string]smapiweb.CompatEntry{
			"author.broken": {Status: smapiweb.StatusBroken, Summary: "Crashes.", BrokeIn: "1.6"},
			"author.ok":     {Status: smapiweb.StatusOK},
		},
		ByNexus: map[int]smapiweb.CompatEntry{
			42: {Status: smapiweb.StatusUnofficial, UnofficialURL: "https://example.com/u", Summary: "Patch."},
		},
	}
	mods := []framework.Mod{
		{Key: "a", Enabled: true, UniqueID: "Author.Broken", Name: "Broken"},
		{Key: "b", Enabled: true, UniqueID: "Author.NexusOnly", Name: "Nexus", UpdateKeys: []string{"Nexus:42"}},
		{Key: "c", Enabled: true, UniqueID: "Author.Ok", Name: "Fine"},
		{Key: "d", Enabled: true, UniqueID: "Author.Unknown", Name: "Unknown"},
		{Key: "e", UniqueID: "Author.Broken", Name: "Switched off"},
	}
	all := matchCompat(idx, mods, false)
	if len(all) != 4 {
		t.Fatalf("all: %+v", all)
	}
	info := matchCompat(idx, mods, true)
	if len(info) != 2 {
		t.Fatalf("non-ok: %+v", info)
	}
	if info[0].Status != smapiweb.StatusBroken || info[0].ID != "smapi:Author.Broken" {
		t.Fatalf("id map: %+v", info[0])
	}
	if info[1].Status != smapiweb.StatusUnofficial || info[1].UnofficialURL != "https://example.com/u" {
		t.Fatalf("nexus map: %+v", info[1])
	}
}

func TestMatchCompatSkipsUnofficialRowOnceInstalledMeetsIt(t *testing.T) {
	idx := smapiweb.CompatIndex{ByID: map[string]smapiweb.CompatEntry{
		"bungus.showitemquality": {Status: smapiweb.StatusUnofficial, UnofficialVersion: "1.1.3-unofficial.1-bungus"},
		"hootless.buslocations":  {Status: smapiweb.StatusUnofficial, UnofficialVersion: "1.2.2-unofficial.1-Xytronix"},
		"author.old":             {Status: smapiweb.StatusUnofficial, UnofficialVersion: "2.0.0-unofficial.1-x"},
	}}
	mods := []framework.Mod{
		{Key: "a", Enabled: true, UniqueID: "bungus.ShowItemQuality", Version: "1.1.3-unofficial.1-bungus"},
		{Key: "b", Enabled: true, UniqueID: "hootless.BusLocations", Version: "2.1.1"},
		{Key: "c", Enabled: true, UniqueID: "author.Old", Version: "1.0.0"},
		{Key: "d", Enabled: true, UniqueID: "author.Old", Version: "not a version"},
	}

	got := matchCompat(idx, mods, true)

	if len(got) != 2 || got[0].Key != "c" || got[1].Key != "d" {
		t.Fatalf("matchCompat() = %+v, want only the older and the unparseable install", got)
	}
}
