package problems

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-smapiweb"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

type listedFakeMeta struct {
	requirements map[int][]nexusmods.Requirement
	// requirementErr fails the batched requirements lookup.
	requirementErr error
	pages          map[int]smapiweb.Page
	pageErr        map[int]error
}

// reqs answers the batched lookup the way NexusPagesOf.Requirements does, and counts the calls.
func (f listedFakeMeta) reqs(calls *int) RequirementsOf {
	return func(_ context.Context, ids []int) (map[int][]nexusmods.Requirement, error) {
		*calls++
		if f.requirementErr != nil {
			return nil, f.requirementErr
		}
		out := map[int][]nexusmods.Requirement{}
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

func (f listedFakeMeta) Lookup(_ context.Context, uniqueID string) ([]smapiweb.Ref, error) {
	for _, ref := range f.pages {
		for _, file := range ref.Downloads {
			for _, im := range file.Mods {
				if mod.Equal(mod.SMAPI(im.UniqueID), mod.SMAPI(uniqueID)) {
					return []smapiweb.Ref{{Site: "Nexus", ID: ref.ID}}, nil
				}
			}
		}
	}
	return nil, nil
}

func (f listedFakeMeta) Page(_ context.Context, id int) (smapiweb.Page, error) {
	if err := f.pageErr[id]; err != nil {
		return smapiweb.Page{}, err
	}
	return f.pages[id], nil
}

func (listedFakeMeta) CheckUpdates(context.Context, meta.UpdateRequest) []smapiweb.UpdateResult {
	return nil
}

func (listedFakeMeta) Collection(context.Context, string, string, int) (nexusmods.Collection, error) {
	return nexusmods.Collection{}, nil
}

func listedDependent() framework.Mod {
	return framework.Mod{
		Key:      "nexus-520-100",
		Enabled:  true,
		UniqueID: "Example.Dependent",
		Name:     "Example Dependent",
	}
}

func listedPage(id int, uniqueID string) smapiweb.Page {
	return smapiweb.Page{
		ID:   id,
		Name: "Requirement Page",
		Downloads: []smapiweb.File{{
			Type: "main",
			Mods: []smapiweb.Mod{{UniqueID: uniqueID, Name: "Requirement"}},
		}},
	}
}

func TestListedRequirementSatisfiedByPageKey(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexusmods.Requirement{520: {{ModID: 11148, Name: "Requirement"}}},
		pages:        map[int]smapiweb.Page{11148: listedPage(11148, "Requirement.Mod")},
	}
	mods := []framework.Mod{listedDependent(), {Key: "nexus-11148-200", Enabled: true}}

	result := listedCheck(fake, mods)

	if len(result.Missing) != 0 {
		t.Fatalf("Missing = %#v, want none", result.Missing)
	}
}

func TestListedRequirementSatisfiedByDatasetUniqueID(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexusmods.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}},
		pages:        map[int]smapiweb.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
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

func TestListedRequirementKeepsImplicitModOutOfCleanup(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexusmods.Requirement{520: {{ModID: 3853, Name: "Content Patcher Animations"}}},
		pages:        map[int]smapiweb.Page{3853: listedPage(3853, "spacechase0.ContentPatcherAnimations")},
	}
	mods := []framework.Mod{
		listedDependent(),
		{Key: "nexus-3853-62104", Enabled: true, Implicit: true, UniqueID: "spacechase0.ContentPatcherAnimations", Name: "Content Patcher Animations"},
	}

	result := listedCheck(fake, mods)

	if len(result.Cleanup) != 0 {
		t.Fatalf("Cleanup = %#v, want none: a switched-on mod's Nexus page requires it", result.Cleanup)
	}
}

func TestListedRequirementMissing(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexusmods.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}},
		pages:        map[int]smapiweb.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
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
		requirements: map[int][]nexusmods.Requirement{520: {{ModID: 14426, Name: "Gender Neutrality Mod Tokens", Notes: "For Gender Neutral Version"}}},
		pages:        map[int]smapiweb.Page{14426: listedPage(14426, "GenderNeutrality.Tokens")},
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
		requirements: map[int][]nexusmods.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}, 521: {{ModID: 1915}}},
		pages:        map[int]smapiweb.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
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
	fake := listedFakeMeta{requirements: map[int][]nexusmods.Requirement{520: {
		{ModID: 2400, Name: "SMAPI"},
		{Name: "SMAPI 4.1.6", URL: "https://github.com/Pathoschild/SMAPI/releases/tag/4.1.6", External: true},
		{Name: "smapi", URL: "https://smapi.io", External: true},
	}}}
	if got := listedCheck(fake, []framework.Mod{listedDependent()}); len(got.Missing) != 0 {
		t.Fatalf("Missing = %#v, want the game's loader skipped", got.Missing)
	}
}

func TestListedOutsideRequirementInstalledIsNotANote(t *testing.T) {
	fake := listedFakeMeta{requirements: map[int][]nexusmods.Requirement{520: {
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
	fake := listedFakeMeta{requirements: map[int][]nexusmods.Requirement{520: {{Name: "Some Tool", URL: "https://example.org", External: true, Notes: "run it first"}}}}
	got := listedCheck(fake, []framework.Mod{listedDependent()})
	if len(got.Missing) != 1 {
		t.Fatalf("Missing = %#v, want one note", got.Missing)
	}
	m := got.Missing[0]
	if !m.External || m.Where != nil || m.ID != mod.NewID(OutsideFormat, "Some Tool") || m.Note != "run it first" || got.Count() != 0 {
		t.Fatalf("note = %#v, count %d", m, got.Count())
	}
}

func TestPageWideOutsideRequirementTheManifestOmitsIsOptional(t *testing.T) {
	fake := listedFakeMeta{requirements: map[int][]nexusmods.Requirement{520: {{Name: "Immersive Farm 2 Remastered", URL: "https://example.org", External: true}}}}
	optionalOutside := func(depID string) bool {
		dependent := listedDependent()
		dependent.Dependencies = []mod.Dependency{{UniqueID: depID}}
		got := listedCheck(fake, []framework.Mod{dependent})
		i := slices.IndexFunc(got.Missing, func(m Missing) bool { return m.External })
		if i < 0 {
			t.Fatalf("Missing = %#v, want the outside note", got.Missing)
		}
		return got.Missing[i].Optional
	}
	if !optionalOutside("Pathoschild.ContentPatcher") {
		t.Fatal("a manifest that declares dependencies without the requirement leaves it optional")
	}
	if optionalOutside("FlashShifter.ImmersiveFarm2Remastered") {
		t.Fatal("a manifest that names the requirement keeps it listed")
	}
}

func TestOptionalListedRequirementIsNotCounted(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexusmods.Requirement{520: {{ModID: 14426, Name: "Gender Neutrality Mod Tokens", Notes: "For Gender Neutral Version"}}},
		pages:        map[int]smapiweb.Page{14426: listedPage(14426, "GenderNeutrality.Tokens")},
	}

	result := listedCheck(fake, []framework.Mod{listedDependent()})

	if len(result.Missing) != 1 || result.Count() != 0 {
		t.Fatalf("Missing = %#v, Count = %d, want one optional row counting 0", result.Missing, result.Count())
	}
}

func TestListedRequirementJudgedAlikeForEveryDependent(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]nexusmods.Requirement{
			520: {{ModID: 1915, Name: "Content Patcher"}},
			521: {{ModID: 1915, Name: "Content Patcher"}},
			522: {{ModID: 1915, Name: "Content Patcher"}},
		},
		pages: map[int]smapiweb.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
	}
	named := framework.Mod{
		Key: "nexus-522-102", Enabled: true, UniqueID: "Example.Third", Name: "Third",
		Dependencies: []mod.Dependency{{UniqueID: "Pathoschild.ContentPatcher", Required: true}},
	}
	mods := []framework.Mod{
		listedDependent(),
		{Key: "nexus-521-101", Enabled: true, UniqueID: "Example.Second", Name: "Second"},
		named,
		{Key: "local-content-patcher", UniqueID: "Pathoschild.ContentPatcher"},
	}

	result := listedCheck(fake, mods)

	var got []string
	for _, m := range result.Missing {
		if m.Listed {
			got = append(got, string(m.DependentID)+" "+m.Reason)
		}
	}
	slices.Sort(got)
	want := []string{"smapi:Example.Dependent disabled", "smapi:Example.Second disabled"}
	if !slices.Equal(got, want) {
		t.Fatalf("listed rows = %v, want %v: the mod whose manifest names the requirement is not owed one", got, want)
	}
}

// overlapFramework matches the mod named by marker and runs its analysis only once the requirements lookup has started.
type overlapFramework struct {
	t        *testing.T
	marker   string
	reqStart <-chan struct{}
	analyzed chan<- struct{}
}

func (f overlapFramework) ID() mod.ID { return mod.ID("framework:Overlap.Test") }
func (f overlapFramework) Matches(m framework.Mod) bool {
	return m.UniqueID == f.marker
}

func (f overlapFramework) Analyze(framework.Input) framework.Findings {
	select {
	case <-f.reqStart:
	case <-time.After(5 * time.Second):
		f.t.Error("the requirements were not being fetched while the analyzers ran")
	}
	close(f.analyzed)
	return framework.Findings{}
}

func TestCheckRunsTheAnalyzersWhileTheRequirementsAreFetched(t *testing.T) {
	started := make(chan struct{})
	analyzed := make(chan struct{})
	framework.Register(overlapFramework{t: t, marker: "Overlap.Probe", reqStart: started, analyzed: analyzed})
	probe := listedDependent()
	probe.UniqueID = "Overlap.Probe"
	fake := listedFakeMeta{}
	reqs := func(context.Context, []int) (map[int][]nexusmods.Requirement, error) {
		close(started)
		select {
		case <-analyzed:
		case <-time.After(5 * time.Second):
			t.Error("the analyzers did not run while the requirements were being fetched")
		}
		return map[int][]nexusmods.Requirement{}, nil
	}
	Check(context.Background(), fake, testEnv, []framework.Mod{probe}, reqs)
}
