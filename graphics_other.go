//go:build !linux

package main

// applyGraphics has nothing to set: the Graphics setting only chooses WebKitGTK's renderer on Linux.
func applyGraphics(string) {}
