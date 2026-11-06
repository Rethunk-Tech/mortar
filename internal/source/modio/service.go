package modio

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/source/tokenaccount"
)

// Account is the mod.io sign-in as Settings shows it.
type Account tokenaccount.Account

// Service keeps the mod.io access token for the frontend: the token goes to the OS keyring, and only the account name goes
// to settings.
type Service struct {
	Store  *settings.Store
	Driver Driver
}

func (s *Service) keeper() tokenaccount.Keeper {
	return tokenaccount.Keeper{
		Store:    s.Store,
		KeyName:  keyName,
		Prompt:   "paste your mod.io access token",
		ErrBad:   ErrBadToken,
		Validate: s.Driver.Validate,
		Name:     func(v *settings.Settings) *string { return &v.ModioName },
	}
}

// Account describes the current sign-in.
func (s *Service) Account() Account { return Account(s.keeper().Account()) }

// SignIn tests the token against mod.io, then keeps it in the keyring. A rejected token stores nothing.
func (s *Service) SignIn(ctx context.Context, token string) (Account, error) {
	a, err := s.keeper().SignIn(ctx, token)
	return Account(a), err
}

// SignOut deletes the token and forgets the account.
func (s *Service) SignOut() (Account, error) {
	a, err := s.keeper().SignOut()
	return Account(a), err
}
