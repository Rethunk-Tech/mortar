//go:build windows

package gog

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func windowsOffline() []string {
	dirs := []string{filepath.Join(`C:\GOG Games`, "Stardew Valley")}
	if pf := os.Getenv("ProgramFiles(x86)"); pf != "" {
		dirs = append(dirs, filepath.Join(pf, "GOG Galaxy", "Games", "Stardew Valley"))
	}
	return dirs
}

// lookupGalaxy is the Galaxy registry read; tests replace it so they never touch the real registry.
var lookupGalaxy = readGalaxyRegistry

func readGalaxyRegistry() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\GOG.com\Games\`+AppID, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	p, _, err := k.GetStringValue("path")
	if err != nil {
		return ""
	}
	return p
}

func galaxyPath() string { return lookupGalaxy() }
