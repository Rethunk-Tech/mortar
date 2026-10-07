package nexussvc

import (
	"context"
	"errors"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussso"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// ErrSignedOut means a Nexus call was made without a signed-in account.
var ErrSignedOut = errors.New("sign in to Nexus Mods in Settings before downloading")

// Authed returns c authenticated as the signed-in account: with its OAuth access token when this build has a
// client id (the API key is then never read), else with its personal API key. ErrSignedOut when there is none.
func Authed(store *settings.Store, c *nexus.Client) (*nexus.Client, error) {
	if store.Get().NexusUserID == 0 {
		return nil, ErrSignedOut
	}
	if nexussso.ClientID == "" {
		return keyClient(c)
	}
	return c.WithBearer(func(ctx context.Context) (string, error) {
		t, err := nexussso.OAuth{ClientID: nexussso.ClientID}.Fresh(ctx)
		if errors.Is(err, secret.ErrNotFound) {
			return "", ErrSignedOut
		}
		return t.Access, err
	}), nil
}
