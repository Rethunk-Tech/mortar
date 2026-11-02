package modio

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Account is the mod.io sign-in as Settings shows it.
type Account struct {
	SignedIn bool   `json:"signedIn"`
	Name     string `json:"name"`
}

// Service keeps the mod.io access token for the frontend: the token goes to the OS keyring, and only the account name
// goes to settings.
type Service struct {
	Store  *settings.Store
	Driver Driver
}

// Account describes the current sign-in.
func (s *Service) Account() Account {
	n := s.Store.Get().ModioName
	return Account{SignedIn: n != "", Name: n}
}

// SignIn tests the token against mod.io, then keeps it in the keyring. A rejected token stores nothing.
func (s *Service) SignIn(ctx context.Context, token string) (Account, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Account{}, errors.New("paste your mod.io access token")
	}
	name, err := s.Driver.Validate(ctx, token)
	if errors.Is(err, ErrBadToken) {
		return Account{}, usererr.Wrap(usererr.Invalid, err)
	}
	if err != nil {
		return Account{}, err
	}
	if err := secret.Set(keyName, token); err != nil {
		return Account{}, err
	}
	return s.setName(name)
}

// SignOut deletes the token and forgets the account.
func (s *Service) SignOut() (Account, error) {
	if err := secret.Delete(keyName); err != nil {
		return Account{}, err
	}
	return s.setName("")
}

func (s *Service) setName(name string) (Account, error) {
	if _, err := s.Store.Update(func(v *settings.Settings) { v.ModioName = name }); err != nil {
		return Account{}, err
	}
	return s.Account(), nil
}
