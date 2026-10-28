// Package secret keeps credentials in the operating system's keyring (Secret Service on Linux, Credential Manager
// on Windows), never in Mortar's data folder. Inside a Flatpak they live in a sealed file or the host's keyring instead (flatpak.go).
package secret

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/sandbox"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/zalando/go-keyring"
)

const service = "mortar"

// ErrNotFound means no secret is stored for the name.
var ErrNotFound = keyring.ErrNotFound

// getTimeout bounds how long a caller waits on a read: Secret Service can hold an unlock prompt open for as long as
// the user takes (or forever, headless), and a caller holding an HTTP request open on it starves every other one.
var getTimeout = 3 * time.Second

// ErrWaitingForUnlock is a read whose keyring prompt is still open. It is not a failure: the read keeps running
// and OnUnlocked fires when it completes.
var ErrWaitingForUnlock = usererr.New(usererr.Locked, "Waiting for you to unlock the keyring.")

// keyringGet is the backend read, replaced in tests.
var (
	keyringGet    = keyring.Get
	keyringSet    = keyring.Set
	keyringDelete = keyring.Delete
)

// OnUnlocked is called after a read that outlived getTimeout completes, so the app can retry what it deferred.
var OnUnlocked = func() {}

type flight struct {
	done  chan struct{}
	value string
	err   error
	// stuck is set once a caller gave up on it; later callers then fail at once instead of each waiting.
	stuck bool
}

var (
	mu       sync.Mutex
	inflight = map[string]*flight{}
	// late holds the values of reads that finished after their caller gave up, for the next Get of that name.
	late = map[string]string{}
)

// Get returns the secret stored under name.
func Get(name string) (string, error) { return GetIn(service, name) }

// GetIn returns the secret another program shares under its own service and name, such as a token a mod
// template's build reads too.
func GetIn(svc, name string) (string, error) {
	key := svc + "\n" + name
	mu.Lock()
	if v, ok := late[key]; ok {
		delete(late, key)
		mu.Unlock()
		return v, nil
	}
	f, running := inflight[key]
	if running && f.stuck {
		mu.Unlock()
		return "", ErrWaitingForUnlock
	}
	if !running {
		f = &flight{done: make(chan struct{})}
		inflight[key] = f
		go run(svc, name, f)
	}
	mu.Unlock()
	select {
	case <-f.done:
		return f.value, explain(f.err)
	case <-time.After(getTimeout):
		mu.Lock()
		select {
		case <-f.done:
			mu.Unlock()
			return f.value, explain(f.err)
		default:
		}
		f.stuck = true
		mu.Unlock()
		return "", ErrWaitingForUnlock
	}
}

func run(svc, name string, f *flight) {
	key := svc + "\n" + name
	var v string
	var err error
	if sandbox.InFlatpak() {
		v, err = flatpakGet(flatpakName(svc, name))
	} else {
		v, err = keyringGet(svc, name)
	}
	mu.Lock()
	f.value, f.err = v, err
	delete(inflight, key)
	wasStuck := f.stuck
	if wasStuck && err == nil {
		late[key] = v
	}
	close(f.done)
	mu.Unlock()
	if wasStuck && err == nil {
		OnUnlocked()
	}
}

// Set stores value under name, replacing any earlier one.
func Set(name, value string) error { return SetIn(service, name, value) }

// SetIn stores value under another program's service and name, as GetIn reads it.
func SetIn(svc, name, value string) error {
	forget(svc, name)
	if sandbox.InFlatpak() {
		return flatpakSet(flatpakName(svc, name), value)
	}
	return explain(keyringSet(svc, name, value))
}

func forget(svc, name string) {
	mu.Lock()
	delete(late, svc+"\n"+name)
	mu.Unlock()
}

// flatpakName is the sealed file's name for a secret: Mortar's own keep their bare name.
func flatpakName(svc, name string) string {
	if svc == service {
		return name
	}
	return svc + "." + name
}

// Delete removes the secret under name; one that is already gone is not an error.
func Delete(name string) error { return DeleteIn(service, name) }

// DeleteIn removes the secret under another program's service and name.
func DeleteIn(svc, name string) error {
	forget(svc, name)
	if sandbox.InFlatpak() {
		return flatpakDelete(flatpakName(svc, name))
	}
	if err := keyringDelete(svc, name); err != nil && !errors.Is(err, ErrNotFound) {
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
