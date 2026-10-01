//go:build windows

package backdrop

import "golang.org/x/sys/windows/registry"

// DesktopWallpaper returns the path of the user's desktop wallpaper, or empty when it cannot be read.
func DesktopWallpaper() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Desktop`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue("Wallpaper")
	if err != nil {
		return ""
	}
	return v
}
