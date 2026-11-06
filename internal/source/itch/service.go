package itch

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source/tokenaccount"
)

// Account is the itch.io sign-in as Settings shows it.
type Account tokenaccount.Account

// Service keeps the itch.io API key for the frontend: the key goes to the OS keyring, and only the account name goes
// to settings.
type Service struct {
	Store  *settings.Store
	Driver Driver
}

func (s *Service) keeper() tokenaccount.Keeper {
	return tokenaccount.Keeper{
		Store:    s.Store,
		KeyName:  keyName,
		Prompt:   "paste your itch.io API key",
		ErrBad:   ErrBadKey,
		Validate: s.Driver.Validate,
		Name:     func(v *settings.Settings) *string { return &v.ItchName },
	}
}

// Account describes the current sign-in.
func (s *Service) Account() Account { return Account(s.keeper().Account()) }

// SignIn tests the key against itch.io, then keeps it in the keyring. A rejected key stores nothing.
func (s *Service) SignIn(ctx context.Context, key string) (Account, error) {
	a, err := s.keeper().SignIn(ctx, key)
	return Account(a), err
}

// SignOut deletes the key and forgets the account.
func (s *Service) SignOut() (Account, error) {
	a, err := s.keeper().SignOut()
	return Account(a), err
}
