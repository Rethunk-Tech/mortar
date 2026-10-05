package itch

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Account is the itch.io sign-in as Settings shows it.
type Account struct {
	SignedIn bool   `json:"signedIn"`
	Name     string `json:"name"`
}

// Service keeps the itch.io API key for the frontend: the key goes to the OS keyring, and only the account name goes
// to settings.
type Service struct {
	Store  *settings.Store
	Driver Driver
}

// Account describes the current sign-in.
func (s *Service) Account() Account {
	n := s.Store.Get().ItchName
	return Account{SignedIn: n != "", Name: n}
}

// SignIn tests the key against itch.io, then keeps it in the keyring. A rejected key stores nothing.
func (s *Service) SignIn(ctx context.Context, key string) (Account, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Account{}, errors.New("paste your itch.io API key")
	}
	name, err := s.Driver.Validate(ctx, key)
	if errors.Is(err, ErrBadKey) {
		return Account{}, usererr.Wrap(usererr.Invalid, err)
	}
	if err != nil {
		return Account{}, err
	}
	if err := secret.Set(keyName, key); err != nil {
		return Account{}, err
	}
	return s.setName(name)
}

// SignOut deletes the key and forgets the account.
func (s *Service) SignOut() (Account, error) {
	if err := secret.Delete(keyName); err != nil {
		return Account{}, err
	}
	return s.setName("")
}

func (s *Service) setName(name string) (Account, error) {
	if _, err := s.Store.Update(func(v *settings.Settings) { v.ItchName = name }); err != nil {
		return Account{}, err
	}
	return s.Account(), nil
}
