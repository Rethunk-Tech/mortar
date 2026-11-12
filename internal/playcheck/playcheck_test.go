package playcheck

import (
	"reflect"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
)

func profileName(id string) string {
	return map[string]string{"p2": "Co-op"}[id]
}

func TestGroups(t *testing.T) {
	t.Parallel()
	page := func(name string) *problems.Ref { return &problems.Ref{PageName: name} }
	updates := func(n int) []problems.Update {
		var out []problems.Update
		for i := range n {
			out = append(out, problems.Update{Key: string(rune('a' + i)), Name: string(rune('A' + i)), Version: "2.0.0"})
		}
		return out
	}
	for _, tc := range []struct {
		name string
		in   Input
		want []Group
	}{
		{name: "nothing wrong", in: Input{ProfileID: "p1"}},
		{
			name: "notes and cosmetic rows are not warnings",
			in: Input{ProfileID: "p1", Problems: problems.Result{
				Missing: []problems.Missing{
					{ID: "smapi:Opt", Optional: true},
					{ID: "smapi:Odd", Reason: "unknown"},
					{ID: "outside:Tool", External: true},
				},
				AssetConflicts: []framework.AssetConflict{{Names: []string{"A"}, Cosmetic: true}},
				Broken:         []problems.Broken{{Name: "Gone", Status: "abandoned"}, {Name: "Odd", Status: "game-unknown"}},
			}},
		},
		{
			name: "a requirement with no page name reads Unknown mod with the id as its title",
			in:   Input{Problems: problems.Result{Missing: []problems.Missing{{ID: "smapi:Need.This"}}}},
			want: []Group{{Kind: "missing", Count: 1, Names: []string{"Unknown mod"}, NameTitles: []string{"Need.This"}}},
		},
		{
			name: "a requirement is named by its page, and listed once for every dependent",
			in: Input{Problems: problems.Result{Missing: []problems.Missing{
				{ID: "smapi:SpaceCore", Where: page("SpaceCore")},
				{ID: "smapi:spacecore", Where: page("SpaceCore")},
			}}},
			want: []Group{{Kind: "missing", Count: 1, Names: []string{"SpaceCore"}}},
		},
		{
			name: "a disabled requirement is named by its installed copy",
			in: Input{
				Problems:  problems.Result{Missing: []problems.Missing{{ID: "smapi:Need.This", Where: page("Page Name")}}},
				Installed: []profile.Mod{{ID: "smapi:need.this", Name: "Need This"}},
			},
			want: []Group{{Kind: "missing", Count: 1, Names: []string{"Need This"}}},
		},
		{
			name: "a listed requirement prefers its page over the installed copy",
			in: Input{
				Problems:  problems.Result{Missing: []problems.Missing{{ID: "smapi:Need.This", Listed: true, Where: page("Page Name")}}},
				Installed: []profile.Mod{{ID: "smapi:Need.This", Name: "Need This"}},
			},
			want: []Group{{Kind: "missing", Count: 1, Names: []string{"Page Name"}}},
		},
		{
			name: "conflicts name their mods apart so the window can word the list",
			in: Input{Problems: problems.Result{AssetConflicts: []framework.AssetConflict{
				{Names: []string{"SVE", "", "Other"}},
				{Target: "maps/town"},
			}}},
			want: []Group{{Kind: "conflicts", Count: 2, Names: []string{"SVE, Other", "maps/town"}, NameParts: [][]string{{"SVE", "Other"}, nil}}},
		},
		{
			name: "updates skip a pinned entry, a skipped source or version, and prereleases when SMAPI's are hidden",
			in: Input{
				ProfileID:  "p1",
				SmapiNever: true,
				Profile: profile.Profile{Entries: []profile.Entry{
					{Key: "pin", Pinned: true},
					{Key: "src", SkipSources: []string{"Nexus"}},
					{Key: "ver", SkipVersion: "2.0.0"},
				}},
				Updates: problems.UpdatesResult{Updates: []problems.Update{
					{Key: "pin", Name: "Pinned", Version: "2.0.0"},
					{Key: "src", Name: "Skipped source", Version: "2.0.0", Source: "Nexus"},
					{Key: "ver", Name: "Skipped version", Version: "2.0.0"},
					{Key: "pre", Name: "Prerelease", Version: "2.0.0", Unofficial: true},
					{Key: "none", Name: "No version"},
					{Key: "ok", Name: "Wanted", Version: "2.0.0", Source: "GitHub"},
				}},
			},
			want: []Group{{Kind: "updates", Count: 1, Names: []string{"Wanted"}}},
		},
		{
			name: "names are capped at five and the count is the full total",
			in:   Input{Updates: problems.UpdatesResult{Updates: updates(6)}},
			want: []Group{{Kind: "updates", Count: 6, Names: []string{"A", "B", "C", "D", "E"}}},
		},
		{
			name: "broken, obsolete and cycle rows and long paths",
			in: Input{Problems: problems.Result{
				Broken:    []problems.Broken{{Name: "Old", Status: "obsolete"}, {Name: "Dead", Status: "abandoned"}, {Name: "Loop", Status: "cycle"}},
				LongPaths: []problems.LongPath{{Name: "Deep"}},
			}},
			want: []Group{
				{Kind: "broken", Count: 2, Names: []string{"Old", "Loop"}},
				{Kind: "longPaths", Count: 1, Names: []string{"Deep"}},
			},
		},
		{
			name: "the newest save was last played with another profile",
			in: Input{ProfileID: "p1", Fit: savessvc.Fit{
				Info: saves.Info{Folder: "Farm_1", Farm: "Sunny", Name: "Sunny World"}, LastProfileID: "p2",
			}},
			want: []Group{{
				Kind: "lastProfile", Count: 1, Names: []string{}, SaveFolder: "Farm_1", SaveFarm: "Sunny", SaveName: "Sunny World",
				ProfileName: "Co-op", SwitchProfileID: "p2",
			}},
		},
		{
			name: "the save's last profile is this one, or was deleted and is named by its id",
			in:   Input{ProfileID: "p1", Fit: savessvc.Fit{LastProfileID: "p1"}},
		},
		{
			name: "the deleted last profile still shows, by id, with nothing to switch to",
			in:   Input{ProfileID: "p1", Fit: savessvc.Fit{LastProfileID: "gone"}},
			want: []Group{{Kind: "lastProfile", Count: 1, Names: []string{}, ProfileName: "gone", SwitchProfileID: "gone"}},
		},
		{
			name: "mods the save used are listed apart: absent ones and switched-off ones",
			in: Input{ProfileID: "p1", Fit: savessvc.Fit{
				LastProfileID: "p1", LastProfileExists: true,
				Missing: []savessvc.Lack{
					{ID: "smapi:Alpha", Name: "Alpha"},
					{ID: "smapi:Beta.Mod", Disabled: true},
					{ID: "smapi:Gamma", Name: "Gamma", Disabled: true},
				},
			}},
			want: []Group{
				{Kind: "saveMods", Count: 1, Names: []string{"Alpha"}, SwitchProfileID: "p1"},
				{Kind: "saveModsOff", Count: 2, Names: []string{"Beta.Mod", "Gamma"}, SwitchProfileID: "p1"},
			},
		},
		{
			name: "changes since the last run keep their count and the first five items",
			in: Input{Changes: &profile.HistoryDiff{Items: []profile.HistoryItem{
				{Kind: "added", Name: "A", Detail: "added A"},
				{Kind: "added", Name: "B", Detail: "added B"},
				{Kind: "added", Name: "C", Detail: "added C"},
				{Kind: "added", Name: "D", Detail: "added D"},
				{Kind: "added", Name: "E", Detail: "added E"},
				{Kind: "added", Name: "F", Detail: "added F"},
			}}},
			want: []Group{{
				Kind: "changes", Count: 6, Names: []string{"added A", "added B", "added C", "added D", "added E"},
				Changes: []profile.HistoryItem{
					{Kind: "added", Name: "A", Detail: "added A"},
					{Kind: "added", Name: "B", Detail: "added B"},
					{Kind: "added", Name: "C", Detail: "added C"},
					{Kind: "added", Name: "D", Detail: "added D"},
					{Kind: "added", Name: "E", Detail: "added E"},
				},
			}},
		},
		{name: "no changes is no group", in: Input{Changes: &profile.HistoryDiff{}}},
	} {
		tc.in.ProfileName = profileName
		got := Groups(tc.in)
		if len(got) == 0 && len(tc.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", tc.name, got, tc.want)
		}
	}
}
