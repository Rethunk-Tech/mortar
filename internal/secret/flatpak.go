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

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/portal"
)

// Inside a Flatpak the Secret Service is not reachable once the sandbox drops its bus name, so secrets live in one
// file in the sandbox's own data folder, sealed with the XDG Secret portal's per-app master secret.

const (
	sealedFile = "secrets"
	plainFile  = "secrets.json"
	hkdfInfo   = "mortar secrets v1"
)

// Replaced in tests.
var (
	portalSecret = portal.RetrieveSecret
	storeDir     = func() string {
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, ".local", "share")
		}
		// Beside Mortar's data folder, never in it, so a backup, move or share of that folder carries no secret.
		return filepath.Join(base, "mortar-secrets")
	}
)

var (
	storeMu sync.Mutex
	// The portal is asked once per process; a failure is cached so a missing portal is not retried on every call.
	keyDone bool
	keyVal  []byte
	keyErr  error
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

// location picks the sealed file when the portal answers, else the plain one. A sealed file with no portal is an
// error: falling back would hide its contents and let the next write start a second store.
// ponytail: a plain file written while the portal was missing is not upgraded to the sealed file later.
func location() (path string, key []byte, err error) {
	dir := storeDir()
	key, kerr := masterKey()
	if kerr == nil {
		return filepath.Join(dir, sealedFile), key, nil
	}
	if _, err := os.Stat(filepath.Join(dir, sealedFile)); err == nil {
		return "", nil, fmt.Errorf("secrets are sealed and the secret portal is unavailable: %w", kerr)
	}
	return filepath.Join(dir, plainFile), nil, nil
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

// save writes atomically: a temp file in the same folder (created 0600), then a rename.
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
	tmp, err := os.CreateTemp(dir, ".secrets-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(b)
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(tmp.Name())
		return werr
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
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
	path, key, err := location()
	if err != nil {
		return err
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
	return update(func(m map[string]string) bool { m[name] = value; return true })
}

func flatpakDelete(name string) error {
	err := update(func(m map[string]string) bool {
		_, ok := m[name]
		delete(m, name)
		return ok
	})
	return err
}

// flatpakGet reads the file store.
func flatpakGet(name string) (string, error) {
	var v string
	var ok bool
	storeMu.Lock()
	path, key, err := location()
	if err == nil {
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
