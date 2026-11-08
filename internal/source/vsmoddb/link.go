package vsmoddb

import "github.com/Rethunk-Tech/go-vintagestory/moddb"

// Schemes is the scheme of the site's 1-click install button.
func (Driver) Schemes() []string { return []string{moddb.Scheme} }

// HandleLinksDefault is false: the game registers these links to install mods itself, so Mortar takes them from the
// system only when the user opts in.
func (Driver) HandleLinksDefault() bool { return false }
