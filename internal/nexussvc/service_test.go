package nexussvc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/secret"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/zalando/go-keyring"
)

func TestSignInOutKeepsKeyOutOfSettings(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Apikey") != "secret-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"user_id":7,"name":"Ada","is_premium":true}`))
	}))
	defer srv.Close()
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	c := nexus.New("1")
	c.BaseURL = srv.URL
	s := NewService(store, c)
	ctx := context.Background()

	if _, err := s.SignIn(ctx, "wrong"); err == nil {
		t.Fatal("bad key accepted")
	}
	if _, err := secret.Get(keyName); !errors.Is(err, secret.ErrNotFound) || s.Account().SignedIn {
		t.Fatalf("rejected key left state behind: %v", err)
	}
	acct, err := s.SignIn(ctx, " secret-key ")
	if err != nil || !acct.SignedIn || acct.Name != "Ada" || !acct.Premium {
		t.Fatalf("sign in = %+v, %v", acct, err)
	}
	if got, _ := secret.Get(keyName); got != "secret-key" {
		t.Fatalf("keyring holds %q", got)
	}
	if acct, err = s.SignOut(); err != nil || acct.SignedIn {
		t.Fatalf("sign out = %+v, %v", acct, err)
	}
	if _, err := secret.Get(keyName); !errors.Is(err, secret.ErrNotFound) {
		t.Fatal("key survived sign out")
	}
}

func TestModNameNeedsASignedInAccount(t *testing.T) {
	keyring.MockInit()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	store, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(store, nexus.New("1"))
	if _, err := s.ModName(context.Background(), 1); !errors.Is(err, ErrSignedOut) {
		t.Fatalf("got %v", err)
	}
}
