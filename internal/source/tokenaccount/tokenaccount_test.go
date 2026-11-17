package tokenaccount

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

var (
	errBad  = errors.New("rejected token")
	errDown = errors.New("source down")
)

func keeper(t *testing.T, validate func(context.Context, string) (string, error)) Keeper {
	t.Helper()
	keyring.MockInit()
	store, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return Keeper{
		Store:    store,
		KeyName:  t.Name(),
		Prompt:   "paste a token",
		ErrBad:   errBad,
		Validate: validate,
		Name:     func(v *settings.Settings) *string { return &v.ItchName },
	}
}

func TestAccountFollowsTheStoredName(t *testing.T) {
	k := keeper(t, func(context.Context, string) (string, error) { return "farmer", nil })
	if a := k.Account(); a.SignedIn || a.Name != "" {
		t.Fatalf("empty = %+v", a)
	}
	if _, err := k.Store.Update(func(v *settings.Settings) { v.ItchName = "farmer" }); err != nil {
		t.Fatal(err)
	}
	if a := k.Account(); !a.SignedIn || a.Name != "farmer" {
		t.Fatalf("named = %+v", a)
	}
}

func TestSignInOut(t *testing.T) {
	var got []string
	k := keeper(t, func(_ context.Context, token string) (string, error) {
		got = append(got, token)
		switch token {
		case "bad":
			return "", errBad
		case "down":
			return "", errDown
		case "k1":
			return "farmer", nil
		default:
			t.Fatalf("unexpected token %q", token)
			return "", nil
		}
	})
	if _, err := k.SignIn(t.Context(), "  "); err == nil || len(got) != 0 {
		t.Fatalf("empty token: %v validated %v", err, got)
	}
	if _, err := k.SignIn(t.Context(), "bad"); !errors.Is(err, errBad) || usererr.KindOf(err) != usererr.Invalid {
		t.Fatalf("rejected: %v", err)
	}
	if a := k.Account(); a.SignedIn {
		t.Fatal("a rejected token signed in")
	}
	if _, err := k.SignIn(t.Context(), "down"); !errors.Is(err, errDown) {
		t.Fatalf("other error: %v", err)
	}
	acct, err := k.SignIn(t.Context(), " k1 ")
	if err != nil || !acct.SignedIn || acct.Name != "farmer" {
		t.Fatalf("sign in = %+v, %v", acct, err)
	}
	if stored, _ := secret.Get(t.Name()); stored != "k1" {
		t.Fatalf("keyring holds %q", stored)
	}
	if acct, err = k.SignOut(); err != nil || acct.SignedIn || acct.Name != "" {
		t.Fatalf("sign out = %+v, %v", acct, err)
	}
	if stored, err := secret.Get(t.Name()); stored != "" || !errors.Is(err, secret.ErrNotFound) {
		t.Fatalf("key survived sign out: %q %v", stored, err)
	}
}

func TestSignInOutSurfaceKeyringErrors(t *testing.T) {
	k := keeper(t, func(context.Context, string) (string, error) { return "farmer", nil })
	keyring.MockInitWithError(errors.New("the keyring is locked"))
	t.Cleanup(keyring.MockInit)
	if _, err := k.SignIn(t.Context(), "k1"); err == nil {
		t.Fatal("sign-in stored a token the keyring refused")
	}
	if a := k.Account(); a.SignedIn {
		t.Fatal("a keyring failure signed in")
	}
	if _, err := k.SignOut(); err == nil {
		t.Fatal("sign-out ignored a keyring failure")
	}
}
