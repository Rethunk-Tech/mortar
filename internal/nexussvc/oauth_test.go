package nexussvc

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/secret"

	"github.com/Rethunk-Tech/go-nexusmods/sso"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/zalando/go-keyring"
)

func TestOAuthBuildSendsBearerAndNeverTheKey(t *testing.T) {
	keyring.MockInit()
	old := ClientID
	ClientID = "cid"
	t.Cleanup(func() { ClientID = old })
	if err := tokenStore.Save(sso.Tokens{Access: "TOK", Refresh: "R", Expires: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"user_id":1,"name":"n","is_premium":false}`))
	}))
	defer srv.Close()

	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(v *settings.Settings) { v.NexusUserID = 1 }); err != nil {
		t.Fatal(err)
	}
	base := nexusmods.New("Mortar", "1")
	base.BaseURL = srv.URL
	c, err := Authed(store, base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Validate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got.Get("Authorization") != "Bearer TOK" || got.Get("Apikey") != "" {
		t.Fatalf("headers %v", got)
	}
	if _, err := (&Service{}).SignIn(t.Context(), "pasted"); err == nil {
		t.Fatal("a pasted key signed in while OAuth is configured")
	}
}

// A grant a 0.3.8 build kept in the keyring reads back, and one with no access token refreshes instead of
// returning an empty bearer.
func TestStoredGrantFormatReadsBack(t *testing.T) {
	keyring.MockInit()
	const stored = `{"access":"A","refresh":"R","expires":"2031-05-06T07:08:09Z"}`
	if err := secret.Set(keyringItem, stored); err != nil {
		t.Fatal(err)
	}
	got, err := tokenStore.Load()
	want := sso.Tokens{Access: "A", Refresh: "R", Expires: time.Date(2031, 5, 6, 7, 8, 9, 0, time.UTC)}
	if err != nil || got != want {
		t.Fatalf("Load = %+v, %v", got, err)
	}
	if err := tokenStore.Save(want); err != nil {
		t.Fatal(err)
	}
	if raw, _ := secret.Get(keyringItem); raw != stored {
		t.Fatalf("saved %s, want %s", raw, stored)
	}
	if err := forgetOAuth(); err != nil {
		t.Fatal(err)
	}
	if _, err := tokenStore.Load(); !errors.Is(err, sso.ErrNoTokens) {
		t.Fatalf("after forgetting: %v", err)
	}
}

func TestStoredGrantWithoutAccessRefreshes(t *testing.T) {
	keyring.MockInit()
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"NEW","refresh_token":"R2","expires_in":3600}`))
	}))
	defer idp.Close()
	if err := secret.Set(keyringItem, `{"access":"","refresh":"R","expires":"2099-01-01T00:00:00Z"}`); err != nil {
		t.Fatal(err)
	}
	o := oauthConfig()
	o.ClientID, o.AuthBase = "cid", idp.URL
	tok, err := o.Fresh(t.Context())
	if err != nil || tok.Access != "NEW" {
		t.Fatalf("Fresh = %+v, %v", tok, err)
	}
	if saved, _ := tokenStore.Load(); saved.Refresh != "R2" {
		t.Fatalf("rotated refresh token not kept: %+v", saved)
	}
}

// Windows refuses a keyring item over 2,560 bytes: the access token is dropped, never the refresh token.
func TestSaveDropsTheAccessTokenWhenTheItemIsTooBig(t *testing.T) {
	keyring.MockInit()
	prev := setItem
	t.Cleanup(func() { setItem = prev })
	var writes []string
	setItem = func(_, v string) error {
		writes = append(writes, v)
		if len(writes) == 1 {
			return secret.ErrTooBig
		}
		return nil
	}
	if err := tokenStore.Save(sso.Tokens{Access: "A", Refresh: "R", Expires: time.Unix(0, 0).UTC()}); err != nil {
		t.Fatal(err)
	}
	if len(writes) != 2 || !strings.Contains(writes[0], `"access":"A"`) || !strings.Contains(writes[1], `"access":""`) || !strings.Contains(writes[1], `"refresh":"R"`) {
		t.Fatalf("writes %q", writes)
	}
}

// The browser tab and the busy-port message are Mortar's wording over the module's neutral defaults.
func TestSignInWordingIsMortars(t *testing.T) {
	o := oauthConfig()
	if o.Addr != "127.0.0.1:51762" || o.DoneText != "Signed in. You can close this tab and return to Mortar." ||
		o.FailText != "Sign-in did not finish. You can close this tab and try again in Mortar." {
		t.Fatalf("config %+v", o)
	}
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", oauthAddr)
	if err != nil {
		t.Skipf("port is not free on this machine: %v", err)
	}
	defer func() { _ = ln.Close() }()
	s := &Service{client: nexusmods.New("Mortar", "1")}
	o.ClientID, o.OpenBrowser = "cid", func(string) error { return nil }
	_, err = s.signInOAuth(t.Context(), o)
	if err == nil || err.Error() != "Port 51762 is in use; close the app using it and try again" {
		t.Fatalf("error %v", err)
	}
}
