package launchsvc

import (
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// chosenGraphics is the catalog choice the graphicsApi setting names for the game, resolved with the profile's
// override; false when the game offers none, the setting is unset or it names a choice the catalog no longer has.
func chosenGraphics(offer *components.Graphics, st settings.Settings, sc settings.Scope, overrides map[string]string) (components.GraphicsChoice, bool) {
	if offer == nil {
		return components.GraphicsChoice{}, false
	}
	return offer.Choice(settings.ResolveAt(st, "graphicsApi", sc, overrides))
}

// graphicsOffer is the game's catalog graphics block, nil when it has none.
func graphicsOffer(gameID string) *components.Graphics {
	info, _ := components.Game(gameID)
	return info.Graphics
}

// graphicsArgs are the launch arguments of the profile's graphics choice, none while it is unset.
func (s *Service) graphicsArgs(g game.Game, profileID string) []string {
	if s.settings == nil {
		return nil
	}
	c, _ := chosenGraphics(graphicsOffer(g.ID()), s.settings.Get(), settings.Scope{Game: g.ID(), Install: installOf(g), Profile: profileID}, launchOverrides(s.profiles, g.ID(), profileID))
	return c.Args
}

// GraphicsAsk says whether Play must ask which graphics API to use before it launches, and whether the game closed
// right after starting last time.
type GraphicsAsk struct {
	Ask            bool `json:"ask"`
	AfterEarlyExit bool `json:"afterEarlyExit"`
}

// GraphicsAsk asks Play's question when the game offers graphics choices and none is set for the profile, or when the
// last run exited at startup under a choice other than the recommended one.
func (s *Service) GraphicsAsk(gameID, profileID string) (GraphicsAsk, error) {
	offer := graphicsOffer(gameID)
	if offer == nil || s.settings == nil {
		return GraphicsAsk{}, nil
	}
	chosen, set := chosenGraphics(offer, s.settings.Get(), settings.Scope{Game: gameID, Profile: profileID}, launchOverrides(s.profiles, gameID, profileID))
	var m graphicsMarker
	if s.profiles != nil && chosen.ID != offer.Recommended {
		var err error
		if m, err = s.readGraphicsMarker(gameID, profileID); err != nil {
			return GraphicsAsk{}, err
		}
	}
	return GraphicsAsk{Ask: !set || m.Pending, AfterEarlyExit: m.Pending}, nil
}
