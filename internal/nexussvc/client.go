package nexussvc

import (
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// ErrSignedOut means a Nexus call was made without a signed-in account.
var ErrSignedOut = errors.New("sign in to Nexus Mods in Settings before downloading")

// Authed returns c using the signed-in account's key from the keyring, or ErrSignedOut.
func Authed(store *settings.Store, c *nexus.Client) (*nexus.Client, error) {
	if store.Get().NexusUserID == 0 {
		return nil, ErrSignedOut
	}
	key, err := secret.Get(keyName)
	if errors.Is(err, secret.ErrNotFound) {
		// The account is remembered but its key is gone (a reset or new keyring): that is signed out, not a failure.
		return nil, ErrSignedOut
	}
	if err != nil {
		return nil, err
	}
	return c.WithKey(key), nil
}
