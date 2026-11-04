package nexussvc

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	nexusmods "github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/go-nexusmods/sso"
)

// SSOEvent carries an SSOState for each stage of a browser sign-in.
const SSOEvent = "nexus:sso"

// SSO sign-in stages sent in SSOState.State after the ones sso.State names.
const (
	SSODone   = "done"
	SSOFailed = "failed"
)

// SSOState is one stage of a sign-in; Error is set with SSOFailed, and Cancelled when the user ended it.
type SSOState struct {
	State     string `json:"state"`
	Error     string `json:"error,omitempty"`
	Cancelled bool   `json:"cancelled,omitempty"`
}

type ssoRun struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

// StartSSO signs in through the browser and returns the account. It streams SSOEvent states and ends when the
// user approves, CancelSSO is called, the wait times out or Nexus refuses. The account is validated before it is kept (OAuth keeps its tokens instead of a key).
func (s *Service) StartSSO(ctx context.Context) (Account, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.run.mu.Lock()
	if s.run.cancel != nil {
		s.run.mu.Unlock()
		return Account{}, usererr.New(usererr.Busy, "a Nexus sign-in is already in progress")
	}
	s.run.cancel = cancel
	s.run.mu.Unlock()
	defer func() {
		s.run.mu.Lock()
		s.run.cancel = nil
		s.run.mu.Unlock()
	}()

	onState := func(st sso.State) { s.emitSSO(SSOState{State: string(st)}) }
	o := s.oauth
	if o.OpenBrowser == nil {
		o.OpenBrowser = s.openBrowser
	}
	o.OnState = onState
	acct, err := s.signInOAuth(runCtx, o)
	if err != nil {
		cancelled := errors.Is(err, context.Canceled)
		if !cancelled {
			slog.Warn("nexus sign-in failed", "err", err)
		}
		s.emitSSO(SSOState{State: SSOFailed, Error: err.Error(), Cancelled: cancelled})
		return Account{}, err
	}
	s.emitSSO(SSOState{State: SSODone})
	return acct, nil
}

// CancelSSO ends a sign-in in progress; it does nothing when none is running.
func (s *Service) CancelSSO() {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	if s.run.cancel != nil {
		s.run.cancel()
	}
}

func (s *Service) openBrowser(url string) error {
	web, err := opener.Web(url)
	if err != nil {
		return err
	}
	if s.App == nil {
		return errors.New("no browser to open")
	}
	return s.App.OpenURL(web)
}

func (s *Service) emitSSO(st SSOState) {
	if s.App != nil {
		s.App.Emit(SSOEvent, st)
	}
}

// signInOAuth authorizes in the browser, checks the access token against Nexus and keeps the grant in the keyring.
func (s *Service) signInOAuth(ctx context.Context, o sso.OAuth) (Account, error) {
	t, err := o.Authorize(ctx)
	if errors.Is(err, sso.ErrPortBusy) {
		return Account{}, portBusy{}
	}
	if err != nil {
		return Account{}, err
	}
	user, err := o.Identify(ctx, t.Access)
	if err != nil {
		return Account{}, nexus.Worded(err)
	}
	if err := tokenStore.Save(t); err != nil {
		return Account{}, err
	}
	return s.signedIn(nexusmods.User{ID: user.ID, Name: user.Name, IsPremium: user.Premium})
}

// signedIn records the account the player signed in as.
func (s *Service) signedIn(user nexusmods.User) (Account, error) {
	return s.update(func(v *settings.Settings) {
		v.NexusUserID, v.NexusName, v.NexusPremium = user.ID, user.Name, user.IsPremium
	})
}

// retiredKeyItem is the keyring item that held a personal API key before Mortar signed in with OAuth.
const retiredKeyItem = "nexus"

// SettleSignIn runs once at startup. It deletes a personal API key an earlier release kept, since Nexus allows none once
// OAuth is in use, and forgets the account that key signed in, so the player sees they need to sign in again. A grant
// kept with no account recorded (settings restored from a backup, or a test session given to a sandbox) names its account.
func SettleSignIn(ctx context.Context, s *Service) {
	retired, err := retireAPIKey()
	if err != nil {
		slog.Warn("nexus: could not retire the personal API key", "err", err)
	}
	if retired && !HasSession() {
		if _, err := s.update(func(v *settings.Settings) { v.NexusUserID, v.NexusName, v.NexusPremium = 0, "", false }); err != nil {
			slog.Warn("nexus: could not forget the account of the retired API key", "err", err)
		}
	}
	if s.Account().SignedIn || !HasSession() {
		return
	}
	t, err := s.oauth.Fresh(ctx)
	if err != nil {
		slog.Warn("nexus: could not refresh the kept sign-in", "err", err)
		return
	}
	user, err := s.oauth.Identify(ctx, t.Access)
	if err != nil {
		slog.Warn("nexus: could not name the kept sign-in's account", "err", err)
		return
	}
	if _, err := s.signedIn(nexusmods.User{ID: user.ID, Name: user.Name, IsPremium: user.Premium}); err != nil {
		slog.Warn("nexus: could not record the kept sign-in's account", "err", err)
	}
}

// retireAPIKey deletes a kept personal API key and says whether there was one.
func retireAPIKey() (bool, error) {
	if _, err := secret.Get(retiredKeyItem); err != nil {
		if errors.Is(err, secret.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if err := secret.Delete(retiredKeyItem); err != nil {
		return false, err
	}
	slog.Info("nexus: deleted the retired personal API key")
	return true, nil
}
