package loadersvc

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// localBridgeEnv names a folder holding a game's bridge build as <game>.zip. It is for the self-test sandbox and tests,
// to run a companion that has no published release yet; the shipped catalog never points at an unpublished repository.
const localBridgeEnv = "MORTAR_LOCAL_BRIDGES"

// localBridge is the bridge component built from <folder>/<game>.zip, with the zip's own hash standing in for a
// release's. ok is false when the variable is unset or the game has no zip there.
func localBridge(gameID string) (component components.Component, zip string, ok bool) {
	dir := os.Getenv(localBridgeEnv)
	if dir == "" {
		return components.Component{}, "", false
	}
	zip = filepath.Join(dir, gameID+".zip")
	sum, err := fsx.SHA256(zip)
	if err != nil {
		return components.Component{}, "", false
	}
	return components.Component{Game: gameID, Name: "bridge", Kind: "bridge", Version: "local", SHA256: sum}, zip, true
}
