// Package winname folds names the way a case-insensitive file system compares them, so two names that are distinct on
// Linux cannot silently alias one file on Windows.
package winname

import (
	"strings"
	"unicode"
)

// Fold is name reduced to the form a case-insensitive file system compares: each rune becomes the smallest of its case
// orbit, so ς and σ, ſ and s, µ and μ share a key, which strings.ToLower does not give them.
func Fold(name string) string {
	return strings.Map(func(r rune) rune {
		least := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			least = min(least, f)
		}
		return least
	}, name)
}
