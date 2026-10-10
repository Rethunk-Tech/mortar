package sharesvc

import (
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// curseforgeUnavailable is why CurseForge cannot be used now, or "" when the receiver has a key.
func curseforgeUnavailable() string {
	e, ok := source.Get(profile.KindCurseForge)
	if !ok {
		return "CurseForge is not available"
	}
	return source.Unavailable(e.Source)
}
