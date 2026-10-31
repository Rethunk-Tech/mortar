package smapimanifest

import (
	"github.com/Rethunk-Tech/go-smapi"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Order is SMAPI's load order of mods (smapi.Resolve), each Folder's name ordering the walk.
func Order(mods []Mod) []loadorder.Row {
	in := make([]smapi.Mod, 0, len(mods))
	for _, m := range mods {
		sm := smapi.Mod{UniqueID: m.UniqueID, Name: m.Name, Folder: m.Folder, ContentPackFor: m.ContentPackFor}
		for _, d := range m.Dependencies {
			sm.Dependencies = append(sm.Dependencies, smapi.Dependency{UniqueID: d.UniqueID, MinimumVersion: d.MinimumVersion, IsRequired: d.Required})
		}
		in = append(in, sm)
	}
	rows := smapi.Resolve(in)
	out := make([]loadorder.Row, len(rows))
	for i, r := range rows {
		out[i] = loadorder.Row{
			Position: r.Position, ID: mod.SMAPI(r.UniqueID), Name: r.Name, Required: ids(r.Required), Optional: ids(r.Optional),
			Dependents: ids(r.Dependents), MissingRequired: ids(r.MissingRequired), Cycle: r.Cycle,
		}
	}
	return out
}

func ids(uniqueIDs []string) []mod.ID {
	if len(uniqueIDs) == 0 {
		return nil
	}
	out := make([]mod.ID, len(uniqueIDs))
	for i, id := range uniqueIDs {
		out[i] = mod.SMAPI(id)
	}
	return out
}
