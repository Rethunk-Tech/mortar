//go:build windows

package gog

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

func windowsGamesDirs() []string {
	dirs := []string{`C:\GOG Games`}
	if pf := os.Getenv("ProgramFiles(x86)"); pf != "" {
		dirs = append(dirs, filepath.Join(pf, "GOG Galaxy", "Games"))
	}
	return dirs
}

// GalaxyDir is GOG Galaxy's install folder from its registry entry, else its default folder, whether or not it exists.
func GalaxyDir() string {
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\GOG.com\GalaxyClient\paths`, registry.QUERY_VALUE); err == nil {
		p, _, err := k.GetStringValue("client")
		_ = k.Close()
		if err == nil && p != "" {
			return filepath.Clean(p)
		}
	}
	if pf := os.Getenv("ProgramFiles(x86)"); pf != "" {
		return filepath.Join(pf, "GOG Galaxy")
	}
	return ""
}

// lookupGalaxy is the Galaxy registry read; tests replace it so they never touch the real registry.
var lookupGalaxy = readGalaxyRegistry

func readGalaxyRegistry(productID string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\GOG.com\Games\`+productID, registry.QUERY_VALUE)
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

func galaxyPath(productID string) string { return lookupGalaxy(productID) }
