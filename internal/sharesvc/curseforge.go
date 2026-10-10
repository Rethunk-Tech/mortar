package sharesvc

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/curseforge"
)

// curseforgeUnavailable is why CurseForge cannot be used now, or "" when the receiver has a key.
func curseforgeUnavailable() string {
	e, ok := source.Get(profile.KindCurseForge)
	if !ok {
		return "CurseForge is not available"
	}
	return source.Unavailable(e.Source)
}

// curseforgeProjectName is a project's name through the registered driver, or "" when it has no key or the call fails.
func curseforgeProjectName(ctx context.Context, project string) string {
	e, ok := source.Get(profile.KindCurseForge)
	d, isDriver := e.Source.(curseforge.Driver)
	if !ok || !isDriver || source.Unavailable(e.Source) != "" {
		return ""
	}
	name, err := d.ProjectName(ctx, project)
	if err != nil {
		return ""
	}
	return name
}
