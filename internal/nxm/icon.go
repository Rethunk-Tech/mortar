//go:build !windows

package nxm

import _ "embed"

//go:embed icons/mortar.png
var iconPNG []byte

//go:embed icons/mortar.svg
var iconSVG []byte

// IconPNG is Mortar's 256 px icon, for callers that hand an icon to the desktop (portal launchers).
func IconPNG() []byte { return iconPNG }
