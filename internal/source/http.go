package source

import "time"

// Limits every searching driver shares.
const (
	PageSize       = 20
	RequestTimeout = 20 * time.Second
	MaxBody        = 16 << 20
)

// UserAgent names Mortar to a site.
func UserAgent(version string) string {
	if version == "" {
		return "Mortar"
	}
	return "Mortar/" + version
}

// FirstPage is the page number a search starts at.
const FirstPage = 1
