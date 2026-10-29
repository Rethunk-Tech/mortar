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

// hostKeyring reports whether the host's Secret Service answers.
func hostKeyring() bool {
	if !hostDone {
		hostDone = true
		_, err := keyringGet(service, hostProbeName)
		hostVal = err == nil || errors.Is(err, ErrNotFound)
	}
	return hostVal
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
	if hostKeyring() {
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
		if err := keyringSet(service, name, v); err != nil {
			return explain(err)
		}
	}
	return fsx.Remove(plain)
}

// Plaintext reports whether secrets are kept in the unencrypted fallback file: inside a Flatpak with neither the
// secret portal nor a host keyring.
func Plaintext() bool {
	if !sandbox.InFlatpak() {
		return false
	}
	storeMu.Lock()
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

func flatpakSet(name, value string) error {
	if host, err := onHost(); err != nil || host {
		if err != nil {
			return err
		}
		return explain(keyringSet(service, name, value))
	}
	return update(func(m map[string]string) bool { m[name] = value; return true })
}

func flatpakDelete(name string) error {
	if host, err := onHost(); err != nil || host {
		if err != nil {
			return err
		}
		if err := keyringDelete(service, name); err != nil && !errors.Is(err, ErrNotFound) {
			return explain(err)
		}
		return nil
	}
	return update(func(m map[string]string) bool {
		_, ok := m[name]
		delete(m, name)
		return ok
	})
}

// onHost resolves the backend and reports whether the host keyring holds the secrets.
func onHost() (bool, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	_, _, host, err := location()
	return host, err
}

// flatpakGet reads the active store.
func flatpakGet(name string) (string, error) {
	var v string
	var ok bool
	storeMu.Lock()
	path, key, host, err := location()
	switch {
	case err != nil:
	case host:
		storeMu.Unlock()
		return keyringGet(service, name)
	default:
		var m map[string]string
		if m, err = load(path, key); err == nil {
			v, ok = m[name]
		}
	}
	storeMu.Unlock()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}
