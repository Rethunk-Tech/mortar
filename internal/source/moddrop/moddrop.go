// Package moddrop is the ModDrop source driver: page links only, the user downloads in the browser.
package moddrop

import (
	"github.com/Rethunk-Tech/mortar/internal/source"
)

type driver struct{}

var _ = source.Register(driver{})

func (driver) ID() string   { return "moddrop" }
func (driver) Name() string { return "ModDrop" }

func (driver) Modes() []source.Acquire { return []source.Acquire{source.Handoff} }

func (driver) Hosts() []string { return []string{"moddrop.com"} }

// ModPageURL is the mod's page under the game's ModDrop key.
func (driver) ModPageURL(gameKey, id string) string {
	return "https://www.moddrop.com/" + gameKey + "/mods/" + id
}
