package bepinex5

import (
	"context"
	"encoding/json"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// Companion is the Mortar BepInEx Bridge.
func (Loader) Companion() bridge.Companion { return bridge.BepInEx }

// Query asks the running game what it has loaded: status (game version, scene, plugins) or plugins. BepInEx keeps one
// config folder per profile, so the state file is found from the profile alone.
func (Loader) Query(ctx context.Context, _ loader.Target, p loader.ProfileView, what string) (json.RawMessage, error) {
	return bridge.Query(ctx, bridge.BepInEx.StatePath(p.Dir), what)
}

// RunningState is the bridge's status: the game version BepInEx's log never names, the scene and the loaded plugins.
func (Loader) RunningState(ctx context.Context, p loader.ProfileView) (loader.Live, error) {
	return loader.QueryLive(ctx, bridge.BepInEx.StatePath(p.Dir))
}

// Send runs a line in the game's own console through the bridge (Valheim's; a game without one answers with an error
// the Console tab shows). The bridge keeps the line's case.
func (Loader) Send(ctx context.Context, _ loader.Target, p loader.ProfileView, command string) (string, error) {
	return "", bridge.Send(ctx, bridge.BepInEx.StatePath(p.Dir), "console "+command)
}
