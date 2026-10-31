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
}
