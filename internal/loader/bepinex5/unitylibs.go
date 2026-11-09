package bepinex5

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/Rethunk-Tech/go-bepinex"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// placeUnityLibs puts the game's Unity base libraries in the profile, from a copy kept in Mortar's cache, so an IL2CPP
// game's first start needs no network. When they cannot be had, BepInEx fetches them itself at launch.
func placeUnityLibs(ctx context.Context, installDir, profileDir string) {
	ver := bepinex.UnityVersion(installDir)
	if ver == "" {
		slog.Warn("bepinex: no Unity version found, BepInEx fetches the Unity base libraries at first start", "dir", installDir)
		return
	}
	base, err := datadir.Dir()
	if err == nil {
		err = bepinex.PlaceUnityLibs(ctx, installDir, profileDir, filepath.Join(base, "cache"))
	}
	if err != nil {
		slog.Warn("bepinex: the Unity base libraries are left for BepInEx to fetch at first start", "unity", ver, "err", err)
		return
	}
	slog.Info("bepinex: placed the Unity base libraries in the profile", "unity", ver)
}
