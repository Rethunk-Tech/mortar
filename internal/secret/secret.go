// Package secret keeps credentials in the operating system's keyring, never in Mortar's data folder. The store
// itself is go-keyring; this package fixes Mortar's constants and gives its errors the wording users see.
package secret

import (
	"errors"
	"time"

	"github.com/Rethunk-Tech/go-keyring"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// The service, HKDF info and folder name where Mortar 0.3.8+ keeps every secret. Changing any of them makes every
// stored key and token unreadable.
const (
	service  = "mortar"
	sealInfo = "mortar secrets v1"
	folder   = "mortar-secrets"
)

var (
	// ErrNotFound means no secret is stored for the name.
	ErrNotFound = keyring.ErrNotFound
	// ErrTooBig means the keyring refused the value as too large (Windows caps a credential at 2,560 bytes).
	ErrTooBig = keyring.ErrTooBig
	// ErrWaitingForUnlock is a read whose keyring prompt is still open. It is not a failure: the read keeps running
	// and OnUnlocked fires when it completes.
	ErrWaitingForUnlock = usererr.New(usererr.Locked, "Waiting for you to unlock the keyring.")
	// OnUnlocked is called after a read that outlived the read timeout completes, so the app can retry what it deferred.
	OnUnlocked = func() {}
)

var store = func() *keyring.Store {
	s, err := keyring.New(keyring.Options{
		Service:        service,
		SealInfo:       sealInfo,
		Folder:         folder,
		SharedServices: []string{"thunderstore"},
		PortalTimeout:  5 * time.Minute,
		OnUnlocked:     func() { OnUnlocked() },
	})
	if err != nil {
		panic(err) // constant options; TestStoreConstants fails first
	}
	return s
}()

// Get returns the secret stored under name.
func Get(name string) (string, error) { v, err := store.Get(name); return v, mapErr(err) }

// GetIn returns the secret another program shares under its own service and name, such as a token a mod
// template's build reads too.
func GetIn(svc, name string) (string, error) { v, err := store.GetIn(svc, name); return v, mapErr(err) }

// Set stores value under name.
func Set(name, value string) error { return mapErr(store.Set(name, value)) }

// SetIn stores a secret under another program's service and name.
func SetIn(svc, name, value string) error { return mapErr(store.SetIn(svc, name, value)) }

// Delete removes the secret under name; one that is already gone is not an error.
func Delete(name string) error { return mapErr(store.Delete(name)) }

// DeleteIn removes the secret under another program's service and name.
func DeleteIn(svc, name string) error { return mapErr(store.DeleteIn(svc, name)) }

// Plaintext reports whether secrets sit in the unencrypted fallback file. It is false while the store is busy or
// cannot be resolved.
func Plaintext() bool { p, _ := store.Plaintext(); return p }

func mapErr(err error) error {
	switch {
	case errors.Is(err, keyring.ErrPending):
		return ErrWaitingForUnlock
	case errors.Is(err, keyring.ErrLocked):
		return usererr.New(usererr.Unknown, "The keyring is locked. Unlock it, then try again.")
	case errors.Is(err, keyring.ErrNoProvider):
		return usererr.New(usererr.Unknown, "No keyring service is running. Install or start gnome-keyring or KWallet, then try again.")
	}
	return err
}
