package control

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

// updateChangelog returns the notes an update crosses for the mod p.ID in p.Source: its GitHub releases or its Nexus
// changelog. Name is the installed version and Value the latest.
func (s *Services) updateChangelog(ctx context.Context, p Params) ([]nexus.Changelog, error) {
	switch p.Source {
	case "github":
		return s.Nexus.UpdateChangelog(ctx, p.Game, 0, p.ID, p.Name, p.Value)
	case "nexus":
		id, err := strconv.Atoi(p.ID)
		if err != nil {
			return nil, fmt.Errorf("nexus mod id %q is not a number", p.ID)
		}
		return s.Nexus.UpdateChangelog(ctx, p.Game, id, "", p.Name, p.Value)
	default:
		return nil, fmt.Errorf("source %q has no changelog", p.Source)
	}
}
