// Package tokenaccount keeps one source's sign-in token: the token goes to the OS keyring, and only the account name
// goes to settings.
package tokenaccount

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Account is a sign-in as Settings shows it.
type Account struct {
	SignedIn bool   `json:"signedIn"`
	Name     string `json:"name"`
}

// Keeper is the sign-in state of one source.
type Keeper struct {
	Store *settings.Store
	// KeyName is the keyring item the token lives under.
	KeyName string
	// Prompt is the message for an empty token.
	Prompt string
	// ErrBad is the driver's error for a rejected token.
	ErrBad error
	// Validate tests a token against the source and returns the account name.
	Validate func(ctx context.Context, token string) (string, error)
	// Name points at the account name field in settings.
	Name func(*settings.Settings) *string
}

// Account describes the current sign-in.
func (k Keeper) Account() Account {
	v := k.Store.Get()
	n := *k.Name(&v)
	return Account{SignedIn: n != "", Name: n}
}

// SignIn tests the token against the source, then keeps it in the keyring. A rejected token stores nothing.
func (k Keeper) SignIn(ctx context.Context, token string) (Account, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Account{}, errors.New(k.Prompt)
	}
	name, err := k.Validate(ctx, token)
	if errors.Is(err, k.ErrBad) {
		return Account{}, usererr.Wrap(usererr.Invalid, err)
	}
	if err != nil {
		return Account{}, err
	}
	if err := secret.Set(k.KeyName, token); err != nil {
		return Account{}, err
	}
	return k.setName(name)
}

// SignOut deletes the token and forgets the account.
func (k Keeper) SignOut() (Account, error) {
	if err := secret.Delete(k.KeyName); err != nil {
		return Account{}, err
	}
	return k.setName("")
}

func (k Keeper) setName(name string) (Account, error) {
	if _, err := k.Store.Update(func(v *settings.Settings) { *k.Name(v) = name }); err != nil {
		return Account{}, err
	}
	return k.Account(), nil
}
