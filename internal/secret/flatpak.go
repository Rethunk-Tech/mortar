package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/portal"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// Inside a Flatpak secrets live in one file in the sandbox's own data folder, sealed with the XDG Secret portal's
// per-app master secret. Without the portal they go to the host's Secret Service (the manifest grants its bus name),
// and with neither to a plain 0600 file.

const (
	sealedFile = "secrets"
	plainFile  = "secrets.json"
	hkdfInfo   = "mortar secrets v1"
)

// Replaced in tests.
var (
	portalSecret = portal.RetrieveSecret
	storeDir     = func() (string, error) {
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
		// Beside Mortar's data folder, never in it, so a backup, move or share of that folder carries no secret.
		return filepath.Join(base, "mortar-secrets"), nil
	}
)

var (
	storeMu sync.Mutex
	// The portal is asked once per process; a failure is cached so a missing portal is not retried on every call.
	keyDone bool
	keyVal  []byte
	keyErr  error
	// The host keyring is probed once per process too.
	hostDone bool
	hostVal  bool
)

// masterKey returns the AES key derived from the portal secret.
func masterKey() ([]byte, error) {
	if keyDone {
		return keyVal, keyErr
	}
	keyDone = true
	secret, err := portalSecret()
	if err != nil {
		keyErr = err
		return nil, err
	}
	keyVal, keyErr = hkdf.Key(sha256.New, secret, nil, hkdfInfo, 32)
	return keyVal, keyErr
}

// hostProbeName is a name no secret uses: asking for it only shows whether a Secret Service answers.
const hostProbeName = ".probe"

// hostKeyring reports whether the host's Secret Service answers. Only a definite "nothing on the bus" is cached as
// no: a locked or dismissed unlock prompt is an error to retry, never a reason to fall back to the plain file.
func hostKeyring() (bool, error) {
	if hostDone {
		return hostVal, nil
	}
	_, err := keyringGet(service, hostProbeName)
	switch {
	case err == nil || errors.Is(err, ErrNotFound):
		hostVal = true
	case noKeyringService(err):
		hostVal = false
	default:
		return false, explain(err)
	}
	hostDone = true
	return hostVal, nil
}

// location picks the sealed file when the portal answers, else the host keyring, else the plain file. A sealed file
// with no portal is an error: falling back would hide its contents and let the next write start a second store.
// host is true when the keyring holds the secrets and path and key are unused.
func location() (path string, key []byte, host bool, err error) {
	dir, err := storeDir()
	if err != nil {
		return "", nil, false, err
	}
	key, kerr := masterKey()
	if kerr == nil {
		if err := sealPlain(dir, key); err != nil {
			return "", nil, false, err
		}
		return filepath.Join(dir, sealedFile), key, false, nil
	}
	if _, err := os.Stat(filepath.Join(dir, sealedFile)); err == nil {
		return "", nil, false, fmt.Errorf("secrets are sealed and the secret portal is unavailable: %w", kerr)
	}
	host, herr := hostKeyring()
	if herr != nil {
		return "", nil, false, herr
	}
	if host {
		return "", nil, true, keyringPlain(dir)
	}
	return filepath.Join(dir, plainFile), nil, false, nil
}

// keyringPlain moves the secrets of a plain file into the host keyring and removes the file, so none is lost or
// left readable on disk. The file stays when any write fails.
func keyringPlain(dir string) error {
	plain := filepath.Join(dir, plainFile)
	if _, err := os.Lstat(plain); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	m, err := load(plain, nil)
	if err != nil {
		return err
	}
	for name, v := range m {
		svc, item := service, name
		if ns, rest, ok := sharedItem(name); ok {
			svc, item = ns, rest
		}
		if err := keyringSet(svc, item, v); err != nil {
			return explain(err)
		}
	}
	return fsx.Remove(plain)
}

// sharedNamespaces are the services other programs read, which the plain file stores as "<service>.<name>".
var sharedNamespaces = []string{"thunderstore"}

func sharedItem(flat string) (svc, name string, ok bool) {
	for _, ns := range sharedNamespaces {
		if rest, found := strings.CutPrefix(flat, ns+"."); found && rest != "" {
			return ns, rest, true
		}
	}
	return "", "", false
}

// Plaintext reports whether secrets are kept in the unencrypted fallback file: inside a Flatpak with neither the
// secret portal nor a host keyring. A store busy with a keyring prompt reports false instead of waiting on it.
func Plaintext() bool {
	if !sandbox.InFlatpak() || !storeMu.TryLock() {
		return false
	}
	defer storeMu.Unlock()
	_, key, host, err := location()
	return err == nil && !host && key == nil
}

// sealPlain moves the secrets of a plain file, written while the portal was missing, into the sealed file once the
// portal answers, so they are neither lost to the sealed store nor left readable on disk.
func sealPlain(dir string, key []byte) error {
	plain, sealed := filepath.Join(dir, plainFile), filepath.Join(dir, sealedFile)
	if _, err := os.Lstat(sealed); !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if _, err := os.Lstat(plain); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	m, err := load(plain, nil)
	if err != nil {
		return err
	}
	if err := save(sealed, key, m); err != nil {
		return err
	}
	return fsx.Remove(plain)
}

// readFile reads through os.Root so the name cannot leave the store folder.
func readFile(path string) ([]byte, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	return root.ReadFile(filepath.Base(path))
}

func load(path string, key []byte) (map[string]string, error) {
	b, err := readFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if key != nil {
		if b, err = unseal(key, b); err != nil {
			return nil, err
		}
	}
	m := map[string]string{}
	return m, json.Unmarshal(b, &m)
}

// secureDir creates dir and holds it to mode 0700: MkdirAll leaves an existing folder's mode alone, and a symlink or
// another user's folder would put the sealed store where someone else can reach it.
func secureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || !ownedByUser(info) {
		return fmt.Errorf("secrets folder %s is a link or belongs to another user", dir)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fsx.Chmod(dir, 0o700)
	}
	return nil
}

// save writes atomically (0600) into the vetted secrets folder.
func save(path string, key []byte, m map[string]string) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if key != nil {
		if b, err = seal(key, b); err != nil {
			return err
		}
	}
	dir := filepath.Dir(path)
	if err := secureDir(dir); err != nil {
		return err
	}
	return datadir.WriteFile(path, b, 0o600)
}

func gcm(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func seal(key, plain []byte) ([]byte, error) {
	g, err := gcm(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return g.Seal(nonce, nonce, plain, nil), nil
}

func unseal(key, sealed []byte) ([]byte, error) {
	g, err := gcm(key)
	if err != nil {
		return nil, err
	}
	if len(sealed) < g.NonceSize() {
		return nil, errors.New("the secrets file is truncated")
	}
	plain, err := g.Open(nil, sealed[:g.NonceSize()], sealed[g.NonceSize():], nil)
	if err != nil {
		return nil, fmt.Errorf("the secrets file cannot be decrypted with this app's portal secret: %w", err)
	}
	return plain, nil
}

// update loads the store, applies fn and saves when fn reports a change.
func update(fn func(m map[string]string) bool) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	path, key, host, err := location()
	if err != nil {
		return err
	}
	if host {
		return errors.New("update called on the host keyring")
	}
	m, err := load(path, key)
	if err != nil {
		return err
	}
	if !fn(m) {
		return nil
	}
	return save(path, key, m)
}

// flatpakSet stores a secret. On the host keyring another program's item sits under its own service and name, as a
// host-side reader (a mod template's build) looks it up; the name an earlier release wrote is removed.
func flatpakSet(svc, name, value string) error {
	host, err := onHost()
	if err != nil {
		return err
	}
	if !host {
		return update(func(m map[string]string) bool {
			m[flatpakName(svc, name)] = value
			delete(m, tombstone(svc, name))
			return true
		})
	}
	if err := keyringSet(svc, name, value); err != nil {
		return explain(err)
	}
	return dropLegacy(svc, name)
}

func flatpakDelete(svc, name string) error {
	host, err := onHost()
	if err != nil {
		return err
	}
	if !host {
		fn := flatpakName(svc, name)
		if err := update(func(m map[string]string) bool {
			_, had := m[fn]
			_, dead := m[tombstone(svc, name)]
			delete(m, fn)
			m[tombstone(svc, name)] = "1"
			return had || !dead
		}); err != nil {
			return err
		}
		return dropHost(svc, name)
	}
	if err := keyringDelete(svc, name); err != nil && !errors.Is(err, ErrNotFound) {
		return explain(err)
	}
	return dropLegacy(svc, name)
}

// dropLegacy removes the item releases before the host-visible naming stored as service "mortar", "<svc>.<name>".
func dropLegacy(svc, name string) error {
	if svc == service {
		return nil
	}
	if err := keyringDelete(service, flatpakName(svc, name)); err != nil && !errors.Is(err, ErrNotFound) {
		return explain(err)
	}
	return nil
}

// onHost resolves the backend and reports whether the host keyring holds the secrets.
func onHost() (bool, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	_, _, host, err := location()
	return host, err
}

// flatpakGet reads the active store; on the host keyring a missing item falls back to the legacy name.
func flatpakGet(svc, name string) (string, error) {
	var v string
	var ok bool
	storeMu.Lock()
	path, key, host, err := location()
	switch {
	case err != nil:
	case host:
		storeMu.Unlock()
		v, err = keyringGet(svc, name)
		if errors.Is(err, ErrNotFound) && svc != service {
			return keyringGet(service, flatpakName(svc, name))
		}
		return v, err
	default:
		var m map[string]string
		if m, err = load(path, key); err == nil {
			v, ok = m[flatpakName(svc, name)]
			if _, dead := m[tombstone(svc, name)]; dead && !ok {
				key = nil // signed out: never adopt a host copy back
			}
		}
	}
	storeMu.Unlock()
	if err != nil {
		return "", err
	}
	if !ok {
		if key != nil {
			return adoptHost(svc, name)
		}
		return "", ErrNotFound
	}
	return v, nil
}

// tombstone is the sealed-store key marking a signed-out secret, so a host-keyring copy that could not be purged
// (locked host) is never adopted back. Set clears it; a store without any reads as an empty set.
func tombstone(svc, name string) string { return "\x00deleted\x00" + flatpakName(svc, name) }

// hostItems is the host-keyring namings of a secret: its own, then the one an earlier release wrote.
func hostItems(svc, name string) [][2]string {
	items := [][2]string{{svc, name}}
	if svc != service {
		items = append(items, [2]string{service, flatpakName(svc, name)})
	}
	return items
}

// hostChecked holds the items already looked up in the host keyring this run: a locked host keyring raises an unlock
// prompt on every lookup, so each item is asked about once and a later run tries again.
var (
	hostCheckedMu sync.Mutex
	hostChecked   = map[string]bool{}
)

// firstHostCheck reports whether this is the run's first host lookup of the item, and marks it checked.
func firstHostCheck(svc, name string) bool {
	hostCheckedMu.Lock()
	defer hostCheckedMu.Unlock()
	k := svc + "\x00" + name
	if hostChecked[k] {
		return false
	}
	hostChecked[k] = true
	return true
}

// hostAbsent reports a host keyring that cannot answer (no Secret Service, or a locked collection or dismissed
// prompt), which counts as the item not being there.
func hostAbsent(err error) bool {
	return noKeyringService(err) || strings.Contains(err.Error(), unlockFailed)
}

// adoptHost moves a secret stored in the host keyring, while the portal was missing, into the sealed file. It looks
// once per item per run; a host that cannot answer is a plain miss.
func adoptHost(svc, name string) (string, error) {
	if !firstHostCheck(svc, name) {
		return "", ErrNotFound
	}
	for _, it := range hostItems(svc, name) {
		v, err := keyringGet(it[0], it[1])
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil && hostAbsent(err) {
			break
		}
		if err != nil {
			return "", explain(err)
		}
		if err := update(func(m map[string]string) bool { m[flatpakName(svc, name)] = v; return true }); err != nil {
			return "", err
		}
		return v, purgeHost(svc, name)
	}
	return "", ErrNotFound
}

// dropHost removes any host-keyring copy of a secret that lives in the sealed file, once per item per run.
func dropHost(svc, name string) error {
	if !firstHostCheck(svc, name) {
		return nil
	}
	return purgeHost(svc, name)
}

func purgeHost(svc, name string) error {
	for _, it := range hostItems(svc, name) {
		err := keyringDelete(it[0], it[1])
		if err == nil || errors.Is(err, ErrNotFound) {
			continue
		}
		if hostAbsent(err) {
			return nil
		}
		return explain(err)
	}
	return nil
}
