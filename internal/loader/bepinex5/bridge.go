package bepinex5

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// Companion is the Mortar BepInEx Bridge.
func (Loader) Companion() bridge.Companion { return bridge.BepInEx }

// Query asks the running game what it has loaded: status (game version, scene, plugins) or plugins. BepInEx keeps one
// config folder per profile, so the state file is found from the profile alone.
func (Loader) Query(ctx context.Context, _ loader.Target, p loader.ProfileView, what string) (json.RawMessage, error) {
	return bridge.Query(ctx, filepath.Join(p.Dir, filepath.FromSlash(bridge.BepInEx.StateFile)), what)
}

// RunningState is the bridge's status: the game version BepInEx's log never names, the scene and the loaded plugins.
func (l Loader) RunningState(ctx context.Context, p loader.ProfileView) (loader.Live, error) {
	raw, err := l.Query(ctx, loader.Target{}, p, "status")
	if err != nil {
		return loader.Live{}, err
	}
	var st bridge.Status
	if err := json.Unmarshal(raw, &st); err != nil {
		return loader.Live{}, err
	}
	return loader.LiveOf(st), nil
}

// Send runs a line in the game's own console through the bridge (Valheim's; a game without one answers with an error
// the Console tab shows). The bridge keeps the line's case.
func (Loader) Send(ctx context.Context, _ loader.Target, p loader.ProfileView, command string) (string, error) {
	return "", bridge.Send(ctx, filepath.Join(p.Dir, filepath.FromSlash(bridge.BepInEx.StateFile)), "console "+command)
}
