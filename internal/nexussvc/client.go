package nexussvc

import (
	"context"
	"errors"

	"github.com/Rethunk-Tech/go-nexusmods/sso"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// ErrSignedOut means a Nexus call was made without a signed-in account; it is a source.ErrSignedOut.
var ErrSignedOut error = signedOut{}

type signedOut struct{}

func (signedOut) Error() string        { return "sign in to Nexus Mods in Settings before downloading" }
func (signedOut) Is(target error) bool { return target == source.ErrSignedOut }

// Authed returns c authenticated as the signed-in account: with its OAuth access token when this build has a
// client id (the API key is then never read), else with its personal API key. ErrSignedOut when there is none.
func Authed(store *settings.Store, c *nexus.Client) (*nexus.Client, error) {
	if store.Get().NexusUserID == 0 {
		return nil, ErrSignedOut
	}
	if ClientID == "" {
		return keyClient(c)
	}
	return c.WithBearer(func(ctx context.Context) (string, error) {
		t, err := oauthConfig().Fresh(ctx)
		if errors.Is(err, sso.ErrNoTokens) || errors.Is(err, sso.ErrRevoked) {
			return "", ErrSignedOut
		}
		return t.Access, err
	}), nil
}
