package bepinex5

import (
	"cmp"
	"context"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-bepinex"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/loader/thunderpack"
)

// ProfileFiles are what InstallPack lays out in each profile's folder (BepInEx's files sit there, not in the game's
// install), with the marker that records the installed version.
func (Loader) ProfileFiles() []string {
	// .doorstop_version and the runtime folders (dotnet, mono, unstripped_corlib) are the pack's, as bepinex.InstallPack
	// lays them out.
	return append(append([]string{"BepInEx", "doorstop_config.ini", "doorstop_libs", ".doorstop_version", markerFile}, bepinex.Proxies()...),
		"dotnet", "mono", "unstripped_corlib")
}

// releases offers only the BepInEx major the catalog names for the game (5 unless it says otherwise): plugins do not
// load across majors, and Mortar starts only BepInEx 5 on Mono and BepInEx 6 on IL2CPP.
func (l Loader) releases(g components.GameInfo) thunderpack.Releases {
	major := strconv.Itoa(cmp.Or(g.LoaderMajor(ID), 5)) + "."
	return thunderpack.Releases{Name: "BepInEx", Index: l.Index, HTTP: l.HTTP, Keep: func(v string) bool { return strings.HasPrefix(v, major) }}
}

// Versions are the releases of the game's BepInEx pack in its community index, newest first.
func (l Loader) Versions(ctx context.Context, g components.GameInfo) ([]string, error) {
	return l.releases(g).Versions(ctx, g)
}

// Latest is the pack's newest release.
func (l Loader) Latest(ctx context.Context, g components.GameInfo) (string, error) {
	return l.releases(g).Latest(ctx, g)
}

// Fetch downloads the game's BepInEx pack zip of version to dst.
func (l Loader) Fetch(ctx context.Context, g components.GameInfo, version, dst string) error {
	return l.releases(g).Fetch(ctx, g, version, dst)
}
