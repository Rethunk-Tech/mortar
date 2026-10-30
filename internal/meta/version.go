package meta

import "github.com/Rethunk-Tech/go-semverish"

// Newer reports whether a is a newer version than b; either failing to parse means not newer.
func Newer(a, b string) bool {
	c, ok := semverish.Compare(a, b)
	return ok && c > 0
}
