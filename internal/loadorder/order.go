// Package loadorder is a profile's load order as its loader computes it.
package loadorder

import "github.com/Rethunk-Tech/mortar/internal/mod"

// Row is one enabled mod, or plugin, in its loader's load order.
type Row struct {
	Position        int      `json:"position"`
	ID              mod.ID   `json:"id"`
	Name            string   `json:"name"`
	Required        []mod.ID `json:"required,omitempty"`
	Optional        []mod.ID `json:"optional,omitempty"`
	Dependents      []mod.ID `json:"dependents,omitempty"`
	MissingRequired []mod.ID `json:"missingRequired,omitempty"`
	Cycle           bool     `json:"cycle,omitempty"`
	// LastRun is a mod the loader's last run loaded at this place rather than where its dependencies predict it.
	LastRun bool `json:"lastRun,omitempty"`
	// Version, Detail (the file or folder the row came from), Official and Default describe a row of a computed list
	// such as Baldur's Gate 3's or Bannerlord's.
	Version  string `json:"version,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Official bool   `json:"official,omitempty"`
	Default  bool   `json:"default,omitempty"`
	// Issues are the problems the loader found with this row.
	Issues []Issue `json:"issues,omitempty"`
	// Folder is the enabled mod folder that provides the row; the profile resolves it to Entry.
	Folder string `json:"-"`
	// Entry is the key of the profile entry whose place in the mod order sets this row's; empty when the order is not
	// the profile's to change.
	Entry string `json:"entry,omitempty"`
}

// Issue is a problem with a row, worded in the frontend from Code and Params.
type Issue struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Params  map[string]string `json:"params,omitempty"`
}
