package control

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// trackingSource accepts the sources that keep a tracked-mods list; only Nexus does.
func trackingSource(source string) error {
	if source != "nexus" {
		return fmt.Errorf("source %q has no tracked mods; pass --source nexus", source)
	}
	return nil
}

func (s *Services) trackedMissing(ctx context.Context, p Params) (any, error) {
	if err := trackingSource(p.Source); err != nil {
		return nil, err
	}
	if s.Nexus == nil {
		return nil, errors.New("nexus is unavailable")
	}
	prof, err := s.resolve(p.Game, p.Profile)
	if err != nil {
		return nil, err
	}
	return s.Nexus.TrackedMissing(ctx, p.Game, prof.ID)
}

// sourceID is the id of the mod src names in source, or empty when src is from another source.
func sourceID(src profile.Source, source string) string {
	switch {
	case source == "nexus" && src.Kind == profile.KindNexus && src.ModID > 0:
		return strconv.Itoa(src.ModID)
	case source == "github" && src.Kind == profile.KindGitHub:
		return src.Repo
	}
	return ""
}
