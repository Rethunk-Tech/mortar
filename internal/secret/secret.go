// Package secret keeps credentials in the operating system's keyring (Secret Service on Linux, Credential Manager
// on Windows), never in Mortar's files.
package secret

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "mortar"

// ErrNotFound means no secret is stored for the name.
var ErrNotFound = keyring.ErrNotFound

// Get returns the secret stored under name.
func Get(name string) (string, error) { return keyring.Get(service, name) }

// Set stores value under name, replacing any earlier one.
func Set(name, value string) error { return keyring.Set(service, name, value) }

// Delete removes the secret under name; one that is already gone is not an error.
func Delete(name string) error {
	if err := keyring.Delete(service, name); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	return nil
}
