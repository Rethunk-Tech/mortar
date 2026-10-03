package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

func TestCheckUpdatesUsesTheNewestFileInTheInstalledGroup(t *testing.T) {
	const id = "EvenMoreSecretWoods"
	rm := fakeMeta{
		compat: map[string]meta.UpdateResult{
			id: {Known: true, Suggested: &meta.Update{Version: "1.2", URL: "https://www.nexusmods.com/stardewvalley/mods/2364"}},
		},
		pages: map[int]meta.Page{2364: {
			Downloads: []meta.File{
				{
					ID:       9545,
					Type:     "MAIN",
					Version:  "1.2",
					FileName: "EvenMoreSecretWoods - Content Patcher Version-2364-1-2.zip",
					Mods:     []meta.Mod{{UniqueID: id, Version: "1.0.0"}},
				},
				{
					ID:       9546,
					Type:     "MAIN",
					Version:  "1.2",
					FileName: "EvenMoreSecretWoods - raw woods.xnb file-2364-1-2.zip",
				},
			},
		}},
	}
	installed := mod("nexus-2364-9545", id, "1.0.0", true)
	installed.UpdateKeys = []string{"Nexus:2364"}

	got := CheckUpdates(context.Background(), rm, Environment{}, []Installed{installed}, false)
	if len(got.Updates) != 0 {
		t.Fatalf("updates = %+v, want none", got.Updates)
	}
}

func TestCheckUpdatesChoosesANewerFileFromTheSameStem(t *testing.T) {
	const id = "Example.Mod"
	rm := fakeMeta{
		compat: map[string]meta.UpdateResult{
			id: {Known: true, Suggested: &meta.Update{Version: "1.2", URL: "https://www.nexusmods.com/stardewvalley/mods/2364"}},
		},
		pages: map[int]meta.Page{2364: {
			Downloads: []meta.File{
				{
					ID:       9544,
					Type:     "OLD_VERSION",
					Version:  "1.1",
					FileName: "Example.Mod-2364-1-1.zip",
					Mods:     []meta.Mod{{UniqueID: id, Version: "1.0.0"}},
				},
				{
					ID:       9545,
					Type:     "MAIN",
					Version:  "1.2",
					FileName: "Example.Mod-2364-1-2.zip",
					Mods:     []meta.Mod{{UniqueID: id, Version: "1.2.0"}},
				},
				{
					ID:       9546,
					Type:     "MAIN",
					Version:  "1.2",
					FileName: "Example.Mod raw woods-2364-1-2.zip",
				},
			},
		}},
	}
	installed := mod("nexus-2364-9544", id, "1.0.0", true)
	installed.UpdateKeys = []string{"Nexus:2364"}

	got := CheckUpdates(context.Background(), rm, Environment{}, []Installed{installed}, false)
	if len(got.Updates) != 1 || got.Updates[0].Version != "1.2" {
		t.Fatalf("updates = %+v, want one same-stem update", got.Updates)
	}
}

func TestCheckUpdatesTrustsSMAPIOverAStaleDatasetPage(t *testing.T) {
	const id = "Example.MachineControlPanel"
	rm := fakeMeta{
		compat: map[string]meta.UpdateResult{
			id: {Known: true, Suggested: &meta.Update{Version: "2.5.0", URL: "https://www.nexusmods.com/stardewvalley/mods/28261"}},
		},
		pages: map[int]meta.Page{28261: {
			Downloads: []meta.File{{
				ID:       179112,
				Type:     "MAIN",
				Version:  "2.4.1",
				FileName: "Machine Control Panel-28261-2-4-1.zip",
				Mods:     []meta.Mod{{UniqueID: id, Version: "2.4.1"}},
			}},
		}},
	}
	installed := mod("nexus-28261-179112", id, "2.4.1", true)
	installed.UpdateKeys = []string{"Nexus:28261"}

	got := CheckUpdates(context.Background(), rm, Environment{}, []Installed{installed}, false)
	if len(got.Updates) != 1 || got.Updates[0].Version != "2.5.0" {
		t.Fatalf("updates = %+v, want 2.5.0", got.Updates)
	}
}
