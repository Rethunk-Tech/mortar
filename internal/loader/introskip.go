package loader

import (
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// IntroSkip is how far a launch takes the game past its start-up on its own.
type IntroSkip string

const (
	// IntroPlay leaves the start-up as the game has it.
	IntroPlay IntroSkip = ""
	// IntroAnimations skips the game's boot animations and cold opens and leaves every choice to the player.
	IntroAnimations IntroSkip = "intro"
	// IntroToMenu also answers the choices the game asks before its menu (Lethal Company's Online or LAN, taking LAN),
	// so a launch nobody watches reaches the main menu.
	IntroToMenu IntroSkip = "menu"
)

// Builtin is a loader that is part of the game itself: nothing is installed for it, so setup has no loader step and
// the window never offers to install, repair or update it.
type Builtin interface {
	Builtin()
}

// IntroSkipper is a loader whose companion can skip the game's intro, reading the request from the profile at launch.
type IntroSkipper interface {
	// ArmIntroSkip sets the request for the next launch of the profile in dir; IntroPlay clears it.
	ArmIntroSkip(dir string, skip IntroSkip) error
}

// IntroSkipFile is the request a bridge reads when the game starts, in the profile's startup folder: "intro" or "menu"
// (IntroSkip). It is a file rather than an environment variable because a Steam launch starts the game through
// `steam -applaunch`, whose child does not inherit Mortar's environment.
const IntroSkipFile = "startup/skip-intro"

// WriteIntroSkip writes a bridge's intro request for the profile in dir, or removes it for IntroPlay.
func WriteIntroSkip(dir string, skip IntroSkip) error {
	path := filepath.Join(dir, filepath.FromSlash(IntroSkipFile))
	return datadir.WriteOrRemove(path, []byte(skip))
}
