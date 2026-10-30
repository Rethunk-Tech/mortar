package settings

import (
	"slices"

	"github.com/Rethunk-Tech/go-desktopenv/sysaccent"
)

// AccentSystem follows the desktop's accent colour.
const AccentSystem = "system"

// accentColors matches frontend/src/theme/accents.ts; the browser extension shows the same colour as the app.
var accentColors = map[string]string{
	"sand":     "#D6B17A",
	"moss":     "#93B86A",
	"copper":   "#D98C5F",
	"sky":      "#79AEDC",
	"rose":     "#E08A9B",
	"lavender": "#A893DE",
	"teal":     "#5DB8AE",
	"slate":    "#94A7BC",
}

func validAccent(name string) bool { return name == AccentSystem || slices.Contains(accents, name) }

// AccentColor returns the hex colour for an accent name, or the default accent's for an unknown one or a system accent
// the desktop does not give.
func AccentColor(name string) string {
	if name == AccentSystem {
		if c, ok := sysaccent.Read(); ok {
			return c
		}
	}
	if c, ok := accentColors[name]; ok {
		return c
	}
	return accentColors[Defaults().Accent]
}
