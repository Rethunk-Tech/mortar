package backdrop

import "github.com/Rethunk-Tech/go-desktopenv/wallpaper"

// DesktopWallpaper returns the path of the user's desktop wallpaper, or empty when it cannot be read.
func DesktopWallpaper() string { return wallpaper.Path() }
