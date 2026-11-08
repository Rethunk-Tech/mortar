package itch

import "github.com/Rethunk-Tech/go-itchio"

// ValidPage reports whether id is a page name as ParsePageURL returns it.
func ValidPage(id string) bool { return itchio.ValidPage(id) }

// PageURL is the canonical address of a page, or "" for an id that is not one.
func PageURL(id string) string { return itchio.PageURL(id) }

// ParsePageURL reads a pasted game page address and returns "user/game".
func ParsePageURL(text string) (string, bool) { return itchio.ParsePageURL(text) }
