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
