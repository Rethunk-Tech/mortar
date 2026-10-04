package settings

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

// AccentColor returns the hex colour for an accent name, or the default accent's for an unknown one.
func AccentColor(name string) string {
	if c, ok := accentColors[name]; ok {
		return c
	}
	return accentColors[Defaults().Accent]
}
