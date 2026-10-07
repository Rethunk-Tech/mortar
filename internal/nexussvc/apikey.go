package nexussvc

// The personal API key path. Every piece of it is in this file, nexus/auth.go and the key box in the Nexus
// settings; remove them together once OAuth ships.

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/nexussso"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const keyName = "nexus"

// errAPIKeyOff means a key was offered to a build that signs in with OAuth.
var errAPIKeyOff = errors.New("this build signs in to Nexus Mods through the browser; personal API keys are not used")

// APIKeyAvailable is whether a pasted personal API key can sign in; it cannot once OAuth is configured.
func (s *Service) APIKeyAvailable() bool { return nexussso.ClientID == "" }

// SignIn checks the key against Nexus, then keeps it in the keyring. A rejected key stores nothing.
func (s *Service) SignIn(ctx context.Context, key string) (Account, error) {
	if nexussso.ClientID != "" {
		return Account{}, errAPIKeyOff
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return Account{}, errors.New("paste your Nexus Mods personal API key")
	}
	user, err := s.client.WithKey(key).Validate(ctx)
	if errors.Is(err, nexus.ErrUnauthorized) {
		return Account{}, usererr.Wrap(usererr.Invalid, err)
	}
	if err != nil {
		return Account{}, err
	}
	if err := secret.Set(keyName, key); err != nil {
		return Account{}, err
	}
	return s.update(func(v *settings.Settings) {
		v.NexusUserID, v.NexusName, v.NexusPremium = user.ID, user.Name, user.IsPremium
	})
}

func forgetKey() error { return secret.Delete(keyName) }

// keyClient is c authenticated with the stored personal API key.
func keyClient(c *nexus.Client) (*nexus.Client, error) {
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
