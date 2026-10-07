// Package secret keeps credentials in the operating system's keyring (Secret Service on Linux, Credential Manager
// on Windows), never in Mortar's files.
package secret

import (
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/zalando/go-keyring"
)

const service = "mortar"

// ErrNotFound means no secret is stored for the name.
var ErrNotFound = keyring.ErrNotFound

// getTimeout bounds a read: Secret Service can wait forever on an unlock prompt nobody can answer (a headless
// session), and a caller holding an HTTP request open on it starves every other request.
const getTimeout = 3 * time.Second

var errKeyringBusy = usererr.New(usererr.Unknown, "The keyring is waiting for an unlock prompt. Unlock it, then try again.")

// getStuck counts reads that timed out and have not returned; while any is, later reads fail at once instead of
// each waiting out the timeout and leaking a goroutine.
var getStuck atomic.Int32

// Get returns the secret stored under name.
func Get(name string) (string, error) {
	if getStuck.Load() > 0 {
		return "", errKeyringBusy
	}
	type result struct {
		v   string
		err error
	}
	done := make(chan result, 1)
	var state atomic.Int32 // 0 running, 1 returned in time, 2 timed out
	go func() {
		v, err := keyring.Get(service, name)
		if !state.CompareAndSwap(0, 1) {
			getStuck.Add(-1)
		}
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		return r.v, explain(r.err)
	case <-time.After(getTimeout):
		if state.CompareAndSwap(0, 2) {
			getStuck.Add(1)
			return "", errKeyringBusy
		}
		r := <-done
		return r.v, explain(r.err)
	}
}

// Set stores value under name, replacing any earlier one.
func Set(name, value string) error { return explain(keyring.Set(service, name, value)) }

// Delete removes the secret under name; one that is already gone is not an error.
func Delete(name string) error {
	if err := keyring.Delete(service, name); err != nil && !errors.Is(err, ErrNotFound) {
		return explain(err)
	}
	return nil
}

// noProvider are the fragments of the D-Bus errors raised when nothing answers as the Secret Service.
var noProvider = []string{
	"org.freedesktop.DBus.Error.ServiceUnknown",
	"org.freedesktop.DBus.Error.NameHasNoOwner",
	"org.freedesktop.DBus.Error.NoServer",
	"couldn't determine address of session bus",
	"dbus-launch",
}

// explain replaces the raw D-Bus failure of a missing Secret Service provider with a sentence the user can act on.
func explain(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, frag := range noProvider {
		if strings.Contains(msg, frag) {
			return usererr.New(usererr.Unknown, "No keyring service is running. Install or start gnome-keyring or KWallet, then try again.")
		}
	}
	return err
}
