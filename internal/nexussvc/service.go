// Package nexussvc exposes the Nexus Mods account to the frontend: the key goes to the OS keyring, and only the
// account's display fields go to settings.
package nexussvc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/components"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/profile"
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

// UntrackAllResult reports the progress of a bulk untrack operation.
type UntrackAllResult struct {
	Untracked       int  `json:"untracked"`
	Remaining       int  `json:"remaining"`
	StoppedForLimit bool `json:"stoppedForLimit"`
}

// Service signs in and out of Nexus Mods.
type Service struct {
	store   *settings.Store
	client  *nexus.Client
	meta    *meta.Client
	seen    *nexus.SeenStore
	prompts *promptStore
	// App is set after application.New so sign-in and sign-out can emit events.
	App *application.App
	// Profiles is the app's profile store, read when untracking only the mods no profile uses.
	Profiles *profile.Store
}

// NewService keeps mod page details in m's cache.
func NewService(store *settings.Store, client *nexus.Client, m *meta.Client) *Service {
	s := &Service{store: store, client: client, meta: m}
	client.SetLimitsHook(func(lim nexus.Limits) {
		if s.App != nil {
			a := s.Account()
			a.Limits = lim
			s.App.Event.Emit(ChangedEvent, a)
		}
	})
	return s
}

// Limits is the budget from the last Nexus response; it does not call the API.
func (s *Service) Limits() nexus.Limits {
	return s.client.Limits()
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

// ModName is the title of a Stardew Valley mod page on Nexus, for showing what a download link is for.
func (s *Service) ModName(ctx context.Context, modID int) (string, error) {
	c, err := Authed(s.store, s.client)
	if err != nil {
		return "", err
	}
	m, err := c.Mod(ctx, modID)
	return m.Name, err
}

func (s *Service) keyed() (*nexus.Client, error) {
	return Authed(s.store, s.client)
}

// Endorse records the signed-in user's endorsement of modID at version.
func (s *Service) Endorse(ctx context.Context, modID int, version string) (string, error) {
	c, err := s.keyed()
	if err != nil {
		return "", err
	}
	status, err := c.Endorse(ctx, modID, version)
	return string(status), err
}

// Abstain withdraws the signed-in user's endorsement of modID at version.
func (s *Service) Abstain(ctx context.Context, modID int, version string) (string, error) {
	c, err := s.keyed()
	if err != nil {
		return "", err
	}
	status, err := c.Abstain(ctx, modID, version)
	return string(status), err
}

// TrackedMods is the signed-in user's tracked list (briefly cached in the Nexus client).
func (s *Service) TrackedMods(ctx context.Context) ([]nexus.TrackedMod, error) {
	c, err := s.keyed()
	if err != nil {
		return nil, err
	}
	return c.TrackedMods(ctx)
}

// trackedFor returns the signed-in user's tracked Nexus mod ids for a Mortar game.
func (s *Service) trackedFor(ctx context.Context, gameID string) ([]int, error) {
	info, ok := components.BundledGame(gameID)
	if !ok || info.Nexus.Domain == "" {
		return nil, fmt.Errorf("game %q has no Nexus domain", gameID)
	}
	mods, err := s.TrackedMods(ctx)
	if err != nil {
		return nil, err
	}
	var ids []int
	for _, mod := range mods {
		if strings.EqualFold(mod.DomainName, info.Nexus.Domain) {
			ids = append(ids, mod.ModID)
		}
	}
	return ids, nil
}

// TrackedCount returns how many of a Mortar game's mods the user tracks on Nexus.
func (s *Service) TrackedCount(ctx context.Context, gameID string) (int, error) {
	ids, err := s.trackedFor(ctx, gameID)
	return len(ids), err
}

// UntrackAll removes a game's tracked mods, optionally retaining mods used by any profile.
func (s *Service) UntrackAll(ctx context.Context, gameID string, onlyNotInProfiles bool) (UntrackAllResult, error) {
	targets, err := s.trackedFor(ctx, gameID)
	if err != nil {
		return UntrackAllResult{}, err
	}
	if onlyNotInProfiles {
		used, err := s.profileModIDs(gameID)
		if err != nil {
			return UntrackAllResult{}, err
		}
		filtered := targets[:0]
		for _, modID := range targets {
			if !used[modID] {
				filtered = append(filtered, modID)
			}
		}
		targets = filtered
	}
	result := UntrackAllResult{Remaining: len(targets)}
	for i, modID := range targets {
		limits := s.client.Limits()
		if limits.Known && (limits.Daily.Remaining <= nexus.LimitFloor || limits.Hourly.Remaining <= nexus.LimitFloor) {
			result.StoppedForLimit = true
			break
		}
		if err := s.Untrack(ctx, modID); err != nil {
			if _, ok := errors.AsType[*nexus.RateLimitError](err); ok {
				result.StoppedForLimit = true
				break
			}
			return result, err
		}
		result.Untracked++
		result.Remaining = len(targets) - i - 1
	}
	return result, nil
}

func (s *Service) profileModIDs(gameID string) (map[int]bool, error) {
	// Without the store every tracked mod would look unused; refuse rather than untrack mods profiles hold.
	if s.Profiles == nil {
		return nil, errors.New("profiles are unavailable")
	}
	profiles, err := s.Profiles.List(gameID)
	if err != nil {
		return nil, err
	}
	used := map[int]bool{}
	for _, p := range profiles {
		if p.Error != "" {
			continue
		}
		for _, entry := range p.Entries {
			if entry.Source.ModID > 0 {
				used[entry.Source.ModID] = true
			}
		}
	}
	return used, nil
}

// Track starts tracking modID for the signed-in user.
func (s *Service) Track(ctx context.Context, modID int) error {
	c, err := s.keyed()
	if err != nil {
		return err
	}
	return c.Track(ctx, modID)
}

// Untrack stops tracking modID for the signed-in user.
func (s *Service) Untrack(ctx context.Context, modID int) error {
	c, err := s.keyed()
	if err != nil {
		return err
	}
	return c.Untrack(ctx, modID)
}
