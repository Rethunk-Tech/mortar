package problems

import (
	"context"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
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

func TestLiveFileIsCurrentSeesANewerFileInTheGroup(t *testing.T) {
	files := []nexus.BatchFile{
		{FileID: 179112, Name: "Machine Control Panel", Version: "2.4.1", Category: "OLD_VERSION"},
		{FileID: 185367, Name: "Machine Control Panel", Version: "2.5.0", Category: "MAIN"},
	}
	installed := mod("nexus-28261-179112", "x", "2.4.1", true)
	if cur, known := liveFileIsCurrent(files, installed, "2.5.0"); cur || !known {
		t.Fatalf("current %v known %v, want an update", cur, known)
	}
}

func TestLiveFileIsCurrentAcceptsAnUnbumpedManifest(t *testing.T) {
	files := []nexus.BatchFile{{FileID: 82663, Name: "Aspen", Version: "0.0.53", Category: "MAIN"}}
	installed := mod("nexus-6754-82663", "invatorzen.AspenCP", "0.0.52", true)
	if cur, known := liveFileIsCurrent(files, installed, "0.0.53"); !cur || !known {
		t.Fatalf("current %v known %v, want current", cur, known)
	}
}

func TestCheckUpdatesUsesOneLiveCallWhenAvailable(t *testing.T) {
	const id = "Example.MachineControlPanel"
	rm := fakeMeta{
		compat: map[string]meta.UpdateResult{
			id: {Known: true, Suggested: &meta.Update{Version: "2.5.0", URL: "https://www.nexusmods.com/stardewvalley/mods/28261"}},
		},
	}
	installed := mod("nexus-28261-179112", id, "2.4.1", true)
	installed.UpdateKeys = []string{"Nexus:28261"}
	calls := 0
	filesOf := func(_ context.Context, ids []int) (map[int][]nexus.BatchFile, error) {
		calls++
		return map[int][]nexus.BatchFile{28261: {
			{FileID: 179112, Name: "Machine Control Panel", Version: "2.4.1", Category: "OLD_VERSION"},
			{FileID: 185367, Name: "Machine Control Panel", Version: "2.5.0", Category: "MAIN"},
		}}, nil
	}
	got := checkUpdates(context.Background(), rm, Environment{}, []Installed{installed}, false, false, filesOf)
	if calls != 1 || len(got.Updates) != 1 {
		t.Fatalf("calls %d updates %+v", calls, got.Updates)
	}
}
