// Package ids makes the random identifiers profiles, history events and tools are keyed by.
package ids

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns 16 random hex characters.
func New() string {
	var raw [8]byte
	_, _ = rand.Read(raw[:]) // never fails since Go 1.24
	return hex.EncodeToString(raw[:])
}
