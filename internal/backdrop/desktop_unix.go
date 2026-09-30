//go:build !windows

package backdrop

// DesktopWallpaper returns the path of the user's desktop wallpaper, or empty when it cannot be read.
func DesktopWallpaper() string { return gnomeWallpaper(execRunner) }
