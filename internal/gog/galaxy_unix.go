//go:build !windows

package gog

func galaxyPath(string) string { return "" }

// GalaxyDir is empty: GOG Galaxy runs only on Windows.
func GalaxyDir() string { return "" }
