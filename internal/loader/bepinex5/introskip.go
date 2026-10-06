package bepinex5

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// IntroSkipFile is the request the Mortar BepInEx Bridge reads when the game starts, in the profile's startup folder:
// "intro" or "menu" (loader.IntroSkip). It is a file rather than an environment variable because a Steam launch
// starts the game through `steam -applaunch`, whose child does not inherit Mortar's environment.
const IntroSkipFile = "startup/skip-intro"

// ArmIntroSkip writes the bridge's intro request for the profile in dir, or removes it.
func (Loader) ArmIntroSkip(dir string, skip loader.IntroSkip) error {
	path := filepath.Join(dir, filepath.FromSlash(IntroSkipFile))
	if skip == loader.IntroPlay {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(path, []byte(skip), 0o600)
}
