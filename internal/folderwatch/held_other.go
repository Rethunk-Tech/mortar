//go:build !windows

package folderwatch

// held is false off Windows, where a growing file always reports its write events.
func held(string) bool { return false }
