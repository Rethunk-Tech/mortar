// Package ids makes the random identifiers profiles, history events and tools are keyed by.
package ids

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// New returns 16 random hex characters.
func New() string {
	var raw [8]byte
	_, _ = rand.Read(raw[:]) // never fails since Go 1.24
	return hex.EncodeToString(raw[:])
}

// Is reports whether s has the shape New returns.
func Is(s string) bool {
	if len(s) != 16 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && s == strings.ToLower(s)
}
