package problems

import (
	"context"
	"errors"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

type listedFakeMeta struct {
	requirements   map[int][]meta.Requirement
	requirementErr map[int]error
	pages          map[int]meta.Page
	pageErr        map[int]error
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

func (f listedFakeMeta) PageRequirements(_ context.Context, _ string, id int) ([]meta.Requirement, error) {
	if err := f.requirementErr[id]; err != nil {
		return nil, err
	}
	return f.requirements[id], nil
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
		requirements: map[int][]meta.Requirement{520: {{ModID: 11148, Name: "Requirement"}}},
		pages:        map[int]meta.Page{11148: listedPage(11148, "Requirement.Mod")},
	}
	mods := []framework.Mod{listedDependent(), {Key: "nexus-11148-200", Enabled: true}}

	result := Check(context.Background(), fake, testEnv, mods)

	if len(result.Missing) != 0 {
		t.Fatalf("Missing = %#v, want none", result.Missing)
	}
}

func TestListedRequirementSatisfiedByDatasetUniqueID(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]meta.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}},
		pages:        map[int]meta.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
	}
	mods := []framework.Mod{
		listedDependent(),
		{Key: "local-content-patcher", Enabled: true, UniqueID: "Pathoschild.ContentPatcher"},
	}

	result := Check(context.Background(), fake, testEnv, mods)

	if len(result.Missing) != 0 {
		t.Fatalf("Missing = %#v, want none", result.Missing)
	}
}

func TestListedRequirementMissing(t *testing.T) {
	fake := listedFakeMeta{
		requirements: map[int][]meta.Requirement{520: {{ModID: 1915, Name: "Content Patcher"}}},
		pages:        map[int]meta.Page{1915: listedPage(1915, "Pathoschild.ContentPatcher")},
	}

	result := Check(context.Background(), fake, testEnv, []framework.Mod{listedDependent()})

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
		requirements: map[int][]meta.Requirement{520: {{ModID: 14426, Name: "Gender Neutrality Mod Tokens", Notes: "For Gender Neutral Version"}}},
		pages:        map[int]meta.Page{14426: listedPage(14426, "GenderNeutrality.Tokens")},
	}

	result := Check(context.Background(), fake, testEnv, []framework.Mod{listedDependent()})

	if len(result.Missing) != 1 || !result.Missing[0].Optional || result.Missing[0].Note != "For Gender Neutral Version" {
		t.Fatalf("Missing = %#v, want optional noted requirement", result.Missing)
	}
}

func TestListedRequirementFetchFailureIsUnknown(t *testing.T) {
	fake := listedFakeMeta{
		requirementErr: map[int]error{520: errors.New("offline")},
	}

	result := Check(context.Background(), fake, testEnv, []framework.Mod{listedDependent()})

	if !result.Unknown {
		t.Fatal("Unknown = false, want true")
	}
	if len(result.Missing) != 0 {
		t.Fatalf("Missing = %#v, want none when requirements could not be read", result.Missing)
	}
}
