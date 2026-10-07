package problems

import (
	"context"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
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
	installed := inst("nexus-2364-9545", id, "1.0.0", true)
	installed.UpdateKeys = []string{"Nexus:2364"}

	got := CheckUpdates(context.Background(), rm, testEnv, []framework.Mod{installed}, false)
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
	installed := inst("nexus-2364-9544", id, "1.0.0", true)
	installed.UpdateKeys = []string{"Nexus:2364"}

	got := CheckUpdates(context.Background(), rm, testEnv, []framework.Mod{installed}, false)
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
	installed := inst("nexus-28261-179112", id, "2.4.1", true)
	installed.UpdateKeys = []string{"Nexus:28261"}

	got := CheckUpdates(context.Background(), rm, testEnv, []framework.Mod{installed}, false)
	if len(got.Updates) != 1 || got.Updates[0].Version != "2.5.0" {
		t.Fatalf("updates = %+v, want 2.5.0", got.Updates)
	}
}

func TestLiveFileIsCurrentSeesANewerFileInTheGroup(t *testing.T) {
	files := []nexus.BatchFile{
		{FileID: 179112, Name: "Machine Control Panel", Version: "2.4.1", Category: "OLD_VERSION"},
		{FileID: 185367, Name: "Machine Control Panel", Version: "2.5.0", Category: "MAIN"},
	}
	installed := inst("nexus-28261-179112", "x", "2.4.1", true)
	if cur, known := liveFileIsCurrent(files, installed, "2.5.0"); cur || !known {
		t.Fatalf("current %v known %v, want an update", cur, known)
	}
}

func TestLiveFileIsCurrentAcceptsAnUnbumpedManifest(t *testing.T) {
	files := []nexus.BatchFile{{FileID: 82663, Name: "Aspen", Version: "0.0.53", Category: "MAIN"}}
	installed := inst("nexus-6754-82663", "invatorzen.AspenCP", "0.0.52", true)
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
	installed := inst("nexus-28261-179112", id, "2.4.1", true)
	installed.UpdateKeys = []string{"Nexus:28261"}
	calls := 0
	filesOf := func(_ context.Context, _ nexus.Title, ids []int) (map[int][]nexus.BatchFile, error) {
		calls++
		return map[int][]nexus.BatchFile{28261: {
			{FileID: 179112, Name: "Machine Control Panel", Version: "2.4.1", Category: "OLD_VERSION"},
			{FileID: 185367, Name: "Machine Control Panel", Version: "2.5.0", Category: "MAIN"},
		}}, nil
	}
	got := checkUpdates(context.Background(), rm, testEnv, []framework.Mod{installed}, false, false, filesOf)
	if calls != 1 || len(got.Updates) != 1 {
		t.Fatalf("calls %d updates %+v", calls, got.Updates)
	}
}

func TestGitHubFallbackOnlyWhenTheUpdateIsNotOnGitHub(t *testing.T) {
	keys := []string{"Nexus:6304", "GitHub:Esca-MMC/DestroyableBushes"}
	if got := githubFallback(keys, "https://www.nexusmods.com/stardewvalley/mods/6304"); got != "Esca-MMC/DestroyableBushes" {
		t.Fatalf("fallback %q", got)
	}
	if got := githubFallback(keys, "https://github.com/Esca-MMC/DestroyableBushes/releases"); got != "" {
		t.Fatalf("a GitHub update needs no fallback, got %q", got)
	}
	if got := githubFallback([]string{"Nexus:6304"}, "https://www.nexusmods.com/stardewvalley/mods/6304"); got != "" {
		t.Fatalf("no GitHub key, got %q", got)
	}
}

// Mod 22743's page holds the main file and an optional bundle pack; SMAPI's suggested version is the page's, which
// is the main file's. The bundle pack's update must download its own newer file, or ask for a pick when it has none.
func TestCheckUpdatesPicksTheBundlePacksOwnFile(t *testing.T) {
	const id = "Morghoula.AlchemistryCCCBL.Easy"
	rm := fakeMeta{compat: map[string]meta.UpdateResult{
		id: {Known: true, Suggested: &meta.Update{Version: "2.0.2", URL: "https://www.nexusmods.com/stardewvalley/mods/22743"}},
	}}
	installed := inst("nexus-22743-178711", id, "2.0.1", true)
	installed.UpdateKeys = []string{"Nexus:22743"}
	page := []nexus.BatchFile{
		{FileID: 175151, Name: "Alchemistry", Version: "2.0.1", Category: "OLD_VERSION"},
		{FileID: 175656, Name: "Alchemistry", Version: "2.0.2", Category: "MAIN"},
		{FileID: 175660, Name: "Alchemistry CC Bundles", Version: "2.0.0", Category: "OLD_VERSION"},
		{FileID: 178711, Name: "Alchemistry CC Bundles", Version: "2.0.1", Category: "OPTIONAL"},
	}
	updates := func(files []nexus.BatchFile) []Update {
		filesOf := func(context.Context, nexus.Title, []int) (map[int][]nexus.BatchFile, error) {
			return map[int][]nexus.BatchFile{22743: files}, nil
		}
		return checkUpdates(context.Background(), rm, testEnv, []framework.Mod{installed}, false, false, filesOf).Updates
	}
	check := func(files []nexus.BatchFile) Update {
		t.Helper()
		got := updates(files)
		if len(got) != 1 {
			t.Fatalf("updates = %+v, want one", got)
		}
		return got[0]
	}
	if got := updates(page); len(got) != 0 {
		t.Errorf("the installed bundle file is its download's newest, so the page's 2.0.2 is not its update: %+v", got)
	}
	archived := slices.Concat(page[:3], []nexus.BatchFile{{FileID: 178711, Name: "Alchemistry CC Bundles", Version: "2.0.1", Category: "ARCHIVED"}})
	if u := check(archived); u.FileID != 0 || !u.PickFile {
		t.Errorf("an archived bundle file with no successor: got file %d pick %v, want a pick on Nexus", u.FileID, u.PickFile)
	}
	newer := slices.Concat(page[:3], []nexus.BatchFile{
		{FileID: 178711, Name: "Alchemistry CC Bundles", Version: "2.0.1", Category: "OLD_VERSION"},
		{FileID: 181000, Name: "Alchemistry CC Bundles", Version: "2.0.2", Category: "OPTIONAL"},
	})
	if u := check(newer); u.FileID != 181000 || u.PickFile || u.Version != "2.0.2" {
		t.Errorf("newer bundle file: got file %d pick %v version %s, want 181000 at 2.0.2", u.FileID, u.PickFile, u.Version)
	}
	older := slices.Concat(page[:3], []nexus.BatchFile{
		{FileID: 178711, Name: "Alchemistry CC Bundles", Version: "2.0.1", Category: "OLD_VERSION"},
		{FileID: 179000, Name: "Alchemistry CC Bundles", Version: "2.0.1.1", Category: "OPTIONAL"},
	})
	if u := check(older); u.FileID != 179000 || u.Version != "2.0.1.1" {
		t.Errorf("the row shows the downloaded file's version, not the page's: %+v", u)
	}
}

// Earthy Cursor: the page says 1.0.1 but its only file is the installed 1.0.0.
func TestCheckUpdatesIgnoresAPageVersionWithoutANewFile(t *testing.T) {
	const id = "Example.EarthyCursor"
	rm := fakeMeta{compat: map[string]meta.UpdateResult{
		id: {Known: true, Suggested: &meta.Update{Version: "1.0.1", URL: "https://www.nexusmods.com/stardewvalley/mods/33076"}},
	}}
	installed := inst("nexus-33076-129465", id, "1.0.0", true)
	installed.UpdateKeys = []string{"Nexus:33076"}
	filesOf := func(context.Context, nexus.Title, []int) (map[int][]nexus.BatchFile, error) {
		return map[int][]nexus.BatchFile{33076: {{FileID: 129465, Name: "Earthy Cursor", Version: "1.0.0", Category: "MAIN"}}}, nil
	}
	if got := checkUpdates(context.Background(), rm, testEnv, []framework.Mod{installed}, false, false, filesOf).Updates; len(got) != 0 {
		t.Fatalf("updates = %+v, want none", got)
	}
}

// A file whose author uploaded a successor updates to that file's id, whatever the page says.
func TestCheckUpdatesFollowsTheReplacedByChain(t *testing.T) {
	const id = "Example.Chain"
	rm := fakeMeta{compat: map[string]meta.UpdateResult{
		id: {Known: true, Suggested: &meta.Update{Version: "3.0", URL: "https://www.nexusmods.com/stardewvalley/mods/5"}},
	}}
	installed := inst("nexus-5-10", id, "1.0", true)
	installed.UpdateKeys = []string{"Nexus:5"}
	filesOf := func(context.Context, nexus.Title, []int) (map[int][]nexus.BatchFile, error) {
		return map[int][]nexus.BatchFile{5: {
			{FileID: 10, Name: "Chain", Version: "1.0", Category: "OLD_VERSION", ReplacedBy: 20},
			{FileID: 20, Name: "Chain", Version: "2.0", Category: "OLD_VERSION", ReplacedBy: 30},
			{FileID: 30, Name: "Chain", Version: "3.0", Category: "MAIN"},
		}}, nil
	}
	got := checkUpdates(context.Background(), rm, testEnv, []framework.Mod{installed}, false, false, filesOf).Updates
	if len(got) != 1 || got[0].FileID != 30 || got[0].Version != "3.0" {
		t.Fatalf("updates = %+v, want file 30", got)
	}
}
