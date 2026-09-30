// Package nexussvc exposes the Nexus Mods account to the frontend: the key goes to the OS keyring, and only the
// account's display fields go to settings.
package nexussvc

import (
	"context"
	"errors"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/secret"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ChangedEvent is emitted with the new Account after a sign-in or sign-out.
const ChangedEvent = "nexus:changed"

const keyName = "nexus"

// Account is the signed-in Nexus account and the rate-limit budget last seen.
type Account struct {
	SignedIn bool         `json:"signedIn"`
	Name     string       `json:"name"`
	Premium  bool         `json:"premium"`
	Limits   nexus.Limits `json:"limits"`
}

// Service signs in and out of Nexus Mods.
type Service struct {
	store  *settings.Store
	client *nexus.Client
	// App is set after application.New so sign-in and sign-out can emit events.
	App *application.App
}

func NewService(store *settings.Store, client *nexus.Client) *Service {
	return &Service{store: store, client: client}
}

// Account describes the current sign-in.
func (s *Service) Account() Account {
	cur := s.store.Get()
	return Account{SignedIn: cur.NexusUserID != 0, Name: cur.NexusName, Premium: cur.NexusPremium, Limits: s.client.Limits()}
}

// SignIn checks the key against Nexus, then keeps it in the keyring. A rejected key stores nothing.
func (s *Service) SignIn(ctx context.Context, key string) (Account, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Account{}, errors.New("paste your Nexus Mods personal API key")
	}
	user, err := s.client.WithKey(key).Validate(ctx)
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

// SignOut deletes the key and forgets the account.
func (s *Service) SignOut() (Account, error) {
	if err := secret.Delete(keyName); err != nil {
		return Account{}, err
	}
	return s.update(func(v *settings.Settings) { v.NexusUserID, v.NexusName, v.NexusPremium = 0, "", false })
}

func (s *Service) update(fn func(*settings.Settings)) (Account, error) {
	if _, err := s.store.Update(fn); err != nil {
		return Account{}, err
	}
	acct := s.Account()
	if s.App != nil {
		s.App.Event.Emit(ChangedEvent, acct)
	}
	return acct, nil
}
