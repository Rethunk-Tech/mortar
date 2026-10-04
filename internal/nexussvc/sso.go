package nexussvc

import (
	"context"
	"errors"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/nexussso"
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

// SSOAvailable is whether this build has a Nexus application slug; the UI hides the button otherwise. OAuth is
// not offered: Nexus documents no way to call the v1 API with its access token.
func (s *Service) SSOAvailable() bool {
	return s.sso.Slug != ""
}

// StartSSO signs in through the browser and returns the account. It streams SSOEvent states and ends when the
// user approves, CancelSSO is called, the wait times out or Nexus refuses. The key is stored and validated exactly
// like a pasted one.
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

	cfg := s.sso
	if cfg.OpenBrowser == nil {
		cfg.OpenBrowser = s.openBrowser
	}
	cfg.OnState = func(st nexussso.State) { s.emitSSO(SSOState{State: string(st)}) }
	key, err := cfg.Run(runCtx)
	var acct Account
	if err == nil {
		acct, err = s.SignIn(runCtx, key)
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
	if s.App == nil {
		return errors.New("no browser to open")
	}
	return s.App.Browser.OpenURL(url)
}

func (s *Service) emitSSO(st SSOState) {
	if s.App != nil {
		s.App.Event.Emit(SSOEvent, st)
	}
}
