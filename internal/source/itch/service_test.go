package itch

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestSignInOutKeepsKeyOutOfSettings(t *testing.T) {
	keyring.MockInit()
	dir := t.TempDir()
	store, err := settings.OpenIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{Store: store, Driver: fake(t)}
	if _, err := s.SignIn(context.Background(), "bad"); err == nil || s.Account().SignedIn {
		t.Fatalf("a rejected key signed in: %v", err)
	}
	acct, err := s.SignIn(context.Background(), " k1 ")
	if err != nil || !acct.SignedIn || acct.Name != "farmer" {
		t.Fatalf("sign in = %+v, %v", acct, err)
	}
	if got, _ := secret.Get(keyName); got != "k1" {
		t.Fatalf("keyring holds %q", got)
	}
	b, err := safefs.ReadFile(filepath.Join(dir, settings.FileName))
	if err != nil || strings.Contains(string(b), "k1") {
		t.Fatalf("settings.json holds the key (%v): %s", err, b)
	}
	if acct, err = s.SignOut(); err != nil || acct.SignedIn {
		t.Fatalf("sign out = %+v, %v", acct, err)
	}
	if got, _ := secret.Get(keyName); got != "" {
		t.Fatal("key survived sign out")
	}
}
