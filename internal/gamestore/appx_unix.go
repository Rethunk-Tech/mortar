//go:build !windows

package gamestore

// appxRoot is empty off Windows: only Windows has app packages.
func appxRoot(string) string { return "" }
