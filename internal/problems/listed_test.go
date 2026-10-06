package problems

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

type listedFakeMeta struct {
	requirements map[int][]nexus.Requirement
	// requirementErr fails the batched requirements lookup.
	requirementErr error
	pages          map[int]meta.Page
	pageErr        map[int]error
}

// reqs answers the batched lookup the way NexusPagesOf.Requirements does, and counts the calls.
func (f listedFakeMeta) reqs(calls *int) RequirementsOf {
	return func(_ context.Context, ids []int) (map[int][]nexus.Requirement, error) {
		*calls++
		if f.requirementErr != nil {
			return nil, f.requirementErr
		}
		out := map[int][]nexus.Requirement{}
		for _, id := range ids {
			if r, ok := f.requirements[id]; ok {
				out[id] = r
			}
		}
		return out, nil
	}
}

func listedCheck(f listedFakeMeta, mods []framework.Mod) Result {
	calls := 0
	return Check(context.Background(), f, testEnv, mods, f.reqs(&calls))
}

func (f listedFakeMeta) Lookup(_ context.Context, uniqueID string) ([]meta.Ref, error) {
	for _, ref := range f.pages {
		for _, file := range ref.Downloads {
			for _, im := range file.Mods {
				if mod.Equal(im.ModID(), mod.SMAPI(uniqueID)) {
					return []meta.Ref{{Site: "Nexus", ID: ref.ID}}, nil
				}
			}
		}
	}
	return nil, nil
}

func (f listedFakeMeta) Page(_ context.Context, id int) (meta.Page, error) {
	if err := f.pageErr[id]; err != nil {
		return meta.Page{}, err
	}
	return f.pages[id], nil
}

func (listedFakeMeta) CheckUpdates(context.Context, meta.UpdateRequest) []meta.UpdateResult {
	return nil
}

func (listedFakeMeta) Collection(context.Context, string, string, int) (meta.Collection, error) {
	return meta.Collection{}, nil
}

func listedDependent() framework.Mod {
	return framework.Mod{
		Key:      "nexus-520-100",
		Enabled:  true,
		UniqueID: "Example.Dependent",
		Name:     "Example Dependent",
	}
}

func listedPage(id int, uniqueID string) meta.Page {
	return meta.Page{
		ID:   id,
		Name: "Requirement Page",
		Downloads: []meta.File{{
			Type: "main",
			Mods: []meta.Mod{{UniqueID: uniqueID, Name: "Requirement"}},
		}},
	}
}

func TestListedRequirementSatisfiedByPageKey(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexus.Requirement{520: {{ModID: 11148, Name: "Requirement"}}},
		pages:        map[int]meta.Page{11148: listedPage(11148, "Requirement.Mod")},
	}
	mods := []framework.Mod{listedDependent(), {Key: "nexus-11148-200", Enabled: true}}

	result := listedCheck(fake, mods)

	if len(result.Missing) != 0 {
		t.Fatalf("Missing = %#v, want none", result.Missing)
	}
}

func TestListedRequirementSatisfiedByDatasetUniqueID(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexus.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}},
		pages:        map[int]meta.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
	}
	mods := []framework.Mod{
		listedDependent(),
		{Key: "local-content-patcher", Enabled: true, UniqueID: "Pathoschild.ContentPatcher"},
	}

	result := listedCheck(fake, mods)

	if len(result.Missing) != 0 {
		t.Fatalf("Missing = %#v, want none", result.Missing)
	}
}

func TestListedRequirementMissing(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexus.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}},
		pages:        map[int]meta.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
	}

	result := listedCheck(fake, []framework.Mod{listedDependent()})

	if len(result.Missing) != 1 {
		t.Fatalf("Missing = %#v, want one item", result.Missing)
	}
	missing := result.Missing[0]
	if missing.ID != "smapi:Pathoschild.ContentPatcher" || missing.Reason != "absent" || !missing.Listed {
		t.Fatalf("Missing = %#v, want listed absent requirement", missing)
	}
}

func TestListedRequirementOptionalNote(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexus.Requirement{520: {{ModID: 14426, Name: "Gender Neutrality Mod Tokens", Notes: "For Gender Neutral Version"}}},
		pages:        map[int]meta.Page{14426: listedPage(14426, "GenderNeutrality.Tokens")},
	}

	result := listedCheck(fake, []framework.Mod{listedDependent()})

	if len(result.Missing) != 1 || !result.Missing[0].Optional || result.Missing[0].Note != "For Gender Neutral Version" {
		t.Fatalf("Missing = %#v, want optional noted requirement", result.Missing)
	}
}

func TestListedRequirementFetchFailureIsUnknown(t *testing.T) {
	fake := listedFakeMeta{
		requirementErr: errors.New("offline"),
	}

	result := listedCheck(fake, []framework.Mod{listedDependent()})

	if !result.Unknown {
		t.Fatal("Unknown = false, want true")
	}
	if len(result.Missing) != 0 {
		t.Fatalf("Missing = %#v, want none when requirements could not be read", result.Missing)
	}
}

func TestListedRequirementsAreOneBatchedLookup(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexus.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}, 521: {{ModID: 1915}}},
		pages:        map[int]meta.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
	}
	other := listedDependent()
	other.Key, other.UniqueID = "nexus-521-1", "Example.Other"
	calls := 0
	Check(context.Background(), fake, testEnv, []framework.Mod{listedDependent(), other}, fake.reqs(&calls))
	if calls != 1 {
		t.Fatalf("lookups = %d, want one for every page", calls)
	}
}

func TestListedLoaderIsNeverMissing(t *testing.T) {
	fake := listedFakeMeta{requirements: map[int][]nexus.Requirement{520: {
		{ModID: 2400, Name: "SMAPI"},
		{Name: "SMAPI 4.1.6", URL: "https://github.com/Pathoschild/SMAPI/releases/tag/4.1.6", External: true},
		{Name: "smapi", URL: "https://smapi.io", External: true},
	}}}
	if got := listedCheck(fake, []framework.Mod{listedDependent()}); len(got.Missing) != 0 {
		t.Fatalf("Missing = %#v, want the game's loader skipped", got.Missing)
	}
}

func TestListedOutsideRequirementInstalledIsNotANote(t *testing.T) {
	fake := listedFakeMeta{requirements: map[int][]nexus.Requirement{520: {
		{Name: "Fashion Sense", URL: "https://github.com/Floogen/FashionSense", External: true},
		{Name: "JsonAssets", URL: "https://spacechase0.com", External: true},
		{Name: "Some Tool", URL: "https://example.org", External: true},
		{Name: "BusLocations", URL: "https://github.com/Entoarox/StardewMods", External: true},
		{Name: "Core", URL: "https://example.org/core", External: true},
	}}}
	installed := []framework.Mod{
		{Key: "nexus-21264-1", Enabled: true, UniqueID: "Ivy.BusLocationsContinued", Name: "BusLocations Continued"},
		{Key: "nexus-1-1", Enabled: true, UniqueID: "Some.CoreTweaks", Name: "Core Tweaks"},
		listedDependent(),
		{Key: "nexus-9969-1", Enabled: true, UniqueID: "PeacefulEnd.FashionSense", Name: "Fashion Sense"},
		{Key: "nexus-1720-1", Enabled: true, UniqueID: "spacechase0.JsonAssets", Name: "Json Assets"},
		{Key: "local-tool", Enabled: false, UniqueID: "Some.Tool", Name: "Some Tool"},
	}
	got := listedCheck(fake, installed)
	var ids []mod.ID
	for _, m := range got.Missing {
		ids = append(ids, m.ID)
	}
	want := []mod.ID{mod.NewID(OutsideFormat, "Some Tool"), mod.NewID(OutsideFormat, "Core")}
	if !slices.Equal(ids, want) {
		t.Fatalf("Missing = %v, want the disabled Some Tool and the too-short Core", ids)
	}
}

func TestListedOutsideRequirementIsANote(t *testing.T) {
	fake := listedFakeMeta{requirements: map[int][]nexus.Requirement{520: {{Name: "Some Tool", URL: "https://example.org", External: true, Notes: "run it first"}}}}
	got := listedCheck(fake, []framework.Mod{listedDependent()})
	if len(got.Missing) != 1 {
		t.Fatalf("Missing = %#v, want one note", got.Missing)
	}
	m := got.Missing[0]
	if !m.External || m.Where != nil || m.ID != mod.NewID(OutsideFormat, "Some Tool") || m.Note != "run it first" || got.Count() != 0 {
		t.Fatalf("note = %#v, count %d", m, got.Count())
	}
}

func TestOptionalListedRequirementIsNotCounted(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexus.Requirement{520: {{ModID: 14426, Name: "Gender Neutrality Mod Tokens", Notes: "For Gender Neutral Version"}}},
		pages:        map[int]meta.Page{14426: listedPage(14426, "GenderNeutrality.Tokens")},
	}

	result := listedCheck(fake, []framework.Mod{listedDependent()})

	if len(result.Missing) != 1 || result.Count() != 0 {
		t.Fatalf("Missing = %#v, Count = %d, want one optional row counting 0", result.Missing, result.Count())
	}
}
