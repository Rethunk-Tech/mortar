package loader

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

// IntroSkipper is a loader whose companion can skip the game's intro, reading the request from the profile at launch.
type IntroSkipper interface {
	// ArmIntroSkip sets the request for the next launch of the profile in dir; IntroPlay clears it.
	ArmIntroSkip(dir string, skip IntroSkip) error
}
