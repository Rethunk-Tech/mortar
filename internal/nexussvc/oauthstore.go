package nexussvc

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods/sso"
	"github.com/Rethunk-Tech/mortar/internal/secret"
)

// Build-time values (-ldflags "-X .../nexussvc.Slug=..."); empty leaves the feature off.
var (
	// Slug is the application slug Nexus issues for legacy SSO.
	Slug string
	// ClientID is the OAuth2 client id Nexus issues; when set it takes precedence over Slug.
	ClientID string
)

const (
	// oauthAddr is the one loopback address the sign-in listens on. Nexus registers the callback as a fixed URI, so
	// there is no fallback port. 51762 sits in the IANA dynamic range, which no service owns.
	oauthAddr   = "127.0.0.1:51762"
	keyringItem = "nexus-oauth"
	doneText    = "Signed in. You can close this tab and return to Mortar."
	failText    = "Sign-in did not finish. You can close this tab and try again in Mortar."
)

// oauthConfig is the OAuth sign-in Mortar registered with Nexus, with its grant kept in the keyring.
func oauthConfig() sso.OAuth {
	return sso.OAuth{ClientID: ClientID, Addr: oauthAddr, DoneText: doneText, FailText: failText, Store: oauthStore{}}
}

// storedTokens is the keyring item's JSON, which releases since 0.3.8 wrote.
type storedTokens struct {
	Access  string    `json:"access"`
	Refresh string    `json:"refresh"`
	Expires time.Time `json:"expires"`
}

// oauthStore keeps the OAuth grant in the OS keyring.
type oauthStore struct{}

var tokenStore oauthStore

// setItem is the keyring write, replaced in tests.
var setItem = secret.Set

func (oauthStore) Load() (sso.Tokens, error) {
	raw, err := secret.Get(keyringItem)
	if errors.Is(err, secret.ErrNotFound) {
		return sso.Tokens{}, sso.ErrNoTokens
	}
	if err != nil {
		return sso.Tokens{}, err
	}
	var t storedTokens
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return sso.Tokens{}, err
	}
	return sso.Tokens{Access: t.Access, Refresh: t.Refresh, Expires: t.Expires}, nil
}

// Save leaves the access token out when the keyring refuses the item as too big (Windows caps it at 2,560
// bytes): a refresh can always replace it, and the refresh token Nexus has just rotated must not be lost.
func (oauthStore) Save(t sso.Tokens) error {
	st := storedTokens{Access: t.Access, Refresh: t.Refresh, Expires: t.Expires}
	b, _ := json.Marshal(st)
	err := setItem(keyringItem, string(b))
	if errors.Is(err, secret.ErrTooBig) {
		st.Access = ""
		b, _ = json.Marshal(st)
		return setItem(keyringItem, string(b))
	}
	return err
}

func forgetOAuth() error { return secret.Delete(keyringItem) }

// portBusy is shown to the player as is, so it keeps its sentence form.
type portBusy struct{}

func (portBusy) Error() string {
	return "Port 51762 is in use; close the app using it and try again"
}
