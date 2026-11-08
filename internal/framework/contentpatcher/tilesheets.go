package contentpatcher

import (
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/cp"
)

// tilesheetCleanups word the packs that only load new map and tilesheet assets that nothing installed uses, or that
// only switched-off mods use.
func tilesheetCleanups(uses []cp.TilesheetUse, unreadable bool) []framework.Cleanup {
	out := []framework.Cleanup{}
	if unreadable {
		return out
	}
	for _, use := range uses {
		if use.Enabled {
			continue
		}
		reason, code, params := "Tilesheets not used by any installed mod", "tilesheetsUnused", map[string]string(nil)
		if len(use.Names) > 0 {
			names := strings.Join(use.Names, ", ")
			reason, code, params = "Used only by switched-off mods: "+names, "tilesheetsOnlyDisabled", map[string]string{"names": names}
		}
		out = append(out, framework.Cleanup{
			Key: use.Key, ID: frameworkID(use.ID), Name: use.Name, Reason: reason, Code: code, Params: params,
		})
	}
	return out
}
