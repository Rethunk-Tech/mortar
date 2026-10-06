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
// The version counts only when the bridge says it is the game's own: otherwise it is Unity's Application.version, which
// a game may never set (Lethal Company's is 0.1), and an older bridge says nothing.
func (l Loader) RunningState(ctx context.Context, p loader.ProfileView) (loader.Live, error) {
	raw, err := l.Query(ctx, loader.Target{}, p, "status")
	if err != nil {
		return loader.Live{}, err
	}
	var st struct {
		GameVersion       string `json:"gameVersion"`
		GameVersionSource string `json:"gameVersionSource"`
		Scene             string `json:"scene"`
		Plugins           []struct {
			GUID string `json:"guid"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(raw, &st); err != nil {
		return loader.Live{}, err
	}
	live := loader.Live{Scene: st.Scene, Plugins: make([]string, 0, len(st.Plugins))}
	if st.GameVersionSource == "game" {
		live.GameVersion = st.GameVersion
	}
	for _, pl := range st.Plugins {
		live.Plugins = append(live.Plugins, pl.GUID)
	}
	return live, nil
}
