package nexussvc

import (
	"context"
	"errors"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/Rethunk-Tech/mortar/internal/settings"

	"github.com/Rethunk-Tech/mortar/internal/nexussso"
)

// SSOEvent carries an SSOState for each stage of a browser sign-in.
const SSOEvent = "nexus:sso"

// SSO sign-in stages sent in SSOState.State.
const (
	SSOWaitingBrowser = "waiting-browser"
	SSOConnected      = "connected"
	SSODone           = "done"
	SSOFailed         = "failed"
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

// SSOAvailable is whether this build has a Nexus OAuth client id or application slug; the UI hides the button
// otherwise. A client id selects OAuth PKCE, a slug alone selects legacy SSO.
func (s *Service) SSOAvailable() bool {
	return s.oauth.ClientID != "" || s.sso.Slug != ""
}

// StartSSO signs in through the browser and returns the account. It streams SSOEvent states and ends when the
// user approves, CancelSSO is called, the wait times out or Nexus refuses. The account is validated before it is kept (OAuth keeps its tokens instead of a key).
func (s *Service) StartSSO(ctx context.Context) (Account, error) {
	if !s.SSOAvailable() {
		return Account{}, errSSOOff
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.run.mu.Lock()
	if s.run.cancel != nil {
		s.run.mu.Unlock()
		return Account{}, errors.New("a Nexus sign-in is already in progress")
	}
	s.run.cancel = cancel
	s.run.mu.Unlock()
	defer func() {
		s.run.mu.Lock()
		s.run.cancel = nil
		s.run.mu.Unlock()
	}()

	onState := func(st nexussso.State) { s.emitSSO(SSOState{State: string(st)}) }
	var acct Account
	var err error
	if s.oauth.ClientID != "" {
		o := s.oauth
		if o.OpenBrowser == nil {
			o.OpenBrowser = s.openBrowser
		}
		o.OnState = onState
		acct, err = s.signInOAuth(runCtx, o)
	} else {
		acct, err = s.signInLegacy(runCtx, onState)
	}
	if err != nil {
		s.emitSSO(SSOState{State: SSOFailed, Error: err.Error(), Cancelled: errors.Is(err, context.Canceled)})
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

var errSSOOff = errors.New("nexus single sign-on is not available in this build")

func (s *Service) openBrowser(url string) error {
	web, err := opener.Web(url)
	if err != nil {
		return err
	}
	if s.App == nil {
		return errors.New("no browser to open")
	}
	return s.App.Browser.OpenURL(web)
}

func (s *Service) emitSSO(st SSOState) {
	if s.App != nil {
		s.App.Event.Emit(SSOEvent, st)
	}
}

// signInLegacy runs the slug-based SSO, whose result is a personal API key.
// Remove with the API-key path once OAuth ships.
func (s *Service) signInLegacy(ctx context.Context, onState func(nexussso.State)) (Account, error) {
	l := s.sso
	if l.OpenBrowser == nil {
		l.OpenBrowser = s.openBrowser
	}
	l.OnState = onState
	key, err := l.Run(ctx)
	if err != nil {
		return Account{}, err
	}
	return s.SignIn(ctx, key)
}

// signInOAuth authorizes in the browser, checks the access token against Nexus and keeps the grant in the keyring.
func (s *Service) signInOAuth(ctx context.Context, o nexussso.OAuth) (Account, error) {
	t, err := o.Authorize(ctx)
	if err != nil {
		return Account{}, err
	}
	user, err := s.client.WithBearer(func(context.Context) (string, error) { return t.Access, nil }).Validate(ctx)
	if err != nil {
		return Account{}, err
	}
	if err := nexussso.Save(t); err != nil {
		return Account{}, err
	}
	return s.update(func(v *settings.Settings) {
		v.NexusUserID, v.NexusName, v.NexusPremium = user.ID, user.Name, user.IsPremium
	})
}
