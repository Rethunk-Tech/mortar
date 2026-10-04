// Package secret keeps credentials in the operating system's keyring (Secret Service on Linux, Credential Manager
// on Windows), never in Mortar's files.
package secret

import (
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/zalando/go-keyring"
)

const service = "mortar"

// ErrNotFound means no secret is stored for the name.
var ErrNotFound = keyring.ErrNotFound

// Get returns the secret stored under name.
func Get(name string) (string, error) {
	v, err := keyring.Get(service, name)
	return v, explain(err)
}

// Set stores value under name, replacing any earlier one.
func Set(name, value string) error { return explain(keyring.Set(service, name, value)) }

// Delete removes the secret under name; one that is already gone is not an error.
func Delete(name string) error {
	if err := keyring.Delete(service, name); err != nil && !errors.Is(err, ErrNotFound) {
		return explain(err)
	}
	return nil
}

// noProvider are the fragments of the D-Bus errors raised when nothing answers as the Secret Service.
var noProvider = []string{
	"org.freedesktop.DBus.Error.ServiceUnknown",
	"org.freedesktop.DBus.Error.NameHasNoOwner",
	"org.freedesktop.DBus.Error.NoServer",
	"couldn't determine address of session bus",
	"dbus-launch",
}

// explain replaces the raw D-Bus failure of a missing Secret Service provider with a sentence the user can act on.
func explain(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, frag := range noProvider {
		if strings.Contains(msg, frag) {
			return usererr.New(usererr.Unknown, "No keyring service is running. Install or start gnome-keyring or KWallet, then try again.")
		}
	}
	return err
}
