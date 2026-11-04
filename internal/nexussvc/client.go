package nexussvc

import (
	"context"
	"errors"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/go-nexusmods/sso"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ErrSignedOut means a Nexus call was made without a signed-in account; it is a source.ErrSignedOut.
var ErrSignedOut = usererr.Wrap(usererr.SignedOut, signedOut{})

type signedOut struct{}

func (signedOut) Error() string        { return "sign in to Nexus Mods in Settings before downloading" }
func (signedOut) Is(target error) bool { return target == source.ErrSignedOut }

// Authed returns c authenticated as the signed-in account with its OAuth access token. ErrSignedOut when there is none.
func Authed(store *settings.Store, c *nexusmods.Client) (*nexusmods.Client, error) {
	if store.Get().NexusUserID == 0 {
		return nil, ErrSignedOut
	}
	// Asked now, not at the first call, so a missing grant or a locked keyring leaves a caller free to use public data.
	if _, err := tokenStore.Load(); errors.Is(err, sso.ErrNoTokens) {
		return nil, ErrSignedOut
	} else if err != nil {
		return nil, err
	}
	return WithSession(c), nil
}

// WithSession returns c sending the kept grant's access token, refreshed when it is due. Its calls fail with
// ErrSignedOut when there is no grant.
func WithSession(c *nexusmods.Client) *nexusmods.Client {
	return c.WithBearer(func(ctx context.Context) (string, error) {
		t, err := oauthConfig().Fresh(ctx)
		if errors.Is(err, sso.ErrNoTokens) || errors.Is(err, sso.ErrRevoked) {
			return "", ErrSignedOut
		}
		return t.Access, err
	})
}

// HasSession is whether a sign-in's grant is kept. It reads the keyring and calls nothing.
func HasSession() bool {
	_, err := tokenStore.Load()
	return err == nil
}
