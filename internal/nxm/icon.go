//go:build !windows

package nxm

import _ "embed"

//go:embed icons/mortar.png
var iconPNG []byte

//go:embed icons/mortar.svg
var iconSVG []byte
