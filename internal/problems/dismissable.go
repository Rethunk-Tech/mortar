package problems

import "github.com/Rethunk-Tech/mortar/internal/mod"

// Dismissable is one counted problem a user can dismiss, in the order the CLI numbers them.
type Dismissable struct {
	Kind         string // "listed", "abandoned", "conflict" or "setting"
	ID           mod.ID
	Field        string
	ConflictKind string
	Target       string
}

// DismissableRows lists the result's dismissable problems: listed requirements, abandoned mods,
// counted conflicts and compatibility settings.
func DismissableRows(r Result) []Dismissable {
	var out []Dismissable
	for _, x := range r.Missing {
		if x.Listed {
			out = append(out, Dismissable{Kind: "listed", ID: x.ID})
		}
	}
	for _, x := range r.Broken {
		if x.Status == "abandoned" {
			out = append(out, Dismissable{Kind: "abandoned", ID: x.ID})
		}
	}
	for _, x := range r.AssetConflicts {
		if !x.Cosmetic && x.Kind != "" {
			out = append(out, Dismissable{Kind: "conflict", ConflictKind: x.Kind, Target: x.Target})
		}
	}
	for _, x := range r.Settings {
		out = append(out, Dismissable{Kind: "setting", ID: x.ID, Field: x.Field})
	}
	return out
}
