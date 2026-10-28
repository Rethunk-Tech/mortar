package secret

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/portal"
)

func TestExplainMapsMissingProvider(t *testing.T) {
	raw := errors.New(`The name org.freedesktop.secrets was not provided by any .service files: org.freedesktop.DBus.Error.ServiceUnknown`)
	got := explain(raw).Error()
	if !strings.Contains(got, "No keyring service is running") || strings.Contains(got, "DBus") {
		t.Fatalf("got %q", got)
	}
	other := errors.New("boom")
	if !errors.Is(explain(other), other) || explain(nil) != nil {
		t.Fatal("unrelated errors must pass through")
	}
}

func TestGetOutlivesTimeoutThenCaches(t *testing.T) {
	prev := getTimeout
	getTimeout = 20 * time.Millisecond
	t.Cleanup(func() { getTimeout = prev })
	release := make(chan struct{})
	var reads atomic.Int32
	keyringGet = func(string, string) (string, error) {
		reads.Add(1)
		<-release
		return "token", nil
	}
	unlocked := make(chan struct{}, 1)
	OnUnlocked = func() { unlocked <- struct{}{} }
	t.Cleanup(func() {
		keyringGet = func(_, name string) (string, error) { return "", errors.New("unset " + name) }
		OnUnlocked = func() {}
	})

	if _, err := Get("k"); !errors.Is(err, ErrWaitingForUnlock) {
		t.Fatalf("want ErrWaitingForUnlock, got %v", err)
	}
	start := time.Now()
	if _, err := Get("k"); !errors.Is(err, ErrWaitingForUnlock) || time.Since(start) >= getTimeout {
		t.Fatalf("a read behind a stuck prompt must fail at once, got %v after %v", err, time.Since(start))
	}
	close(release)
	select {
	case <-unlocked:
	case <-time.After(5 * time.Second):
		t.Fatal("no unlock notification")
	}
	if v, err := Get("k"); err != nil || v != "token" {
		t.Fatalf("late value: %q, %v", v, err)
	}
	if reads.Load() != 1 {
		t.Fatalf("the late result must be served without another backend read, got %d reads", reads.Load())
	}
}

// flatpakEnv puts the package in Flatpak mode with a temp store and a fake portal.
func flatpakEnv(t *testing.T, portal func() ([]byte, error)) (dir string) {
	t.Helper()
	t.Setenv("FLATPAK_ID", "tech.rethunk.Mortar")
	dir = filepath.Join(t.TempDir(), "mortar")
	prevPortal, prevDir := portalSecret, storeDir
	prevGet, prevSet, prevDel := keyringGet, keyringSet, keyringDelete
	portalSecret, storeDir = portal, func() string { return dir }
	setHostRing(nil)
	t.Cleanup(func() {
		portalSecret, storeDir = prevPortal, prevDir
		keyringGet, keyringSet, keyringDelete = prevGet, prevSet, prevDel
		keyDone, hostDone = false, false
	})
	return dir
}

// setHostRing stubs the host Secret Service with ring, or as unreachable when ring is nil, and re-probes it.
func setHostRing(ring map[string]string) {
	hostDone = false
	keyDone = false
	if ring == nil {
		down := errors.New("org.freedesktop.DBus.Error.ServiceUnknown")
		keyringGet = func(string, string) (string, error) { return "", down }
		keyringSet = func(string, string, string) error { return down }
		keyringDelete = func(string, string) error { return down }
		return
	}
	keyringGet = func(_, name string) (string, error) {
		if v, ok := ring[name]; ok {
			return v, nil
		}
		return "", ErrNotFound
	}
	keyringSet = func(_, name, v string) error { ring[name] = v; return nil }
	keyringDelete = func(_, name string) error { delete(ring, name); return nil }
}

func fixedPortal(secret string) func() ([]byte, error) {
	return func() ([]byte, error) { return []byte(secret), nil }
}

func TestFlatpakStore(t *testing.T) {
	t.Run("sealed round trip", func(t *testing.T) {
		dir := flatpakEnv(t, fixedPortal("master"))
		if err := Set("nexus", "key-123"); err != nil {
			t.Fatal(err)
		}
		if v, err := Get("nexus"); err != nil || v != "key-123" {
			t.Fatalf("got %q, %v", v, err)
		}
		raw, err := readFile(filepath.Join(dir, "secrets"))
		if err != nil || bytes.Contains(raw, []byte("key-123")) {
			t.Fatalf("the file must be sealed: %v", err)
		}
		if err := Delete("nexus"); err != nil {
			t.Fatal(err)
		}
		if _, err := Get("nexus"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("no portal falls back to a plain 0600 file", func(t *testing.T) {
		dir := flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no such interface") })
		if err := Set("itch", "tok"); err != nil {
			t.Fatal(err)
		}
		if v, err := Get("itch"); err != nil || v != "tok" {
			t.Fatalf("got %q, %v", v, err)
		}
		fi, err := os.Stat(filepath.Join(dir, "secrets.json"))
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("plain file: %v %v", fi, err)
		}
		if di, _ := os.Stat(dir); di.Mode().Perm() != 0o700 {
			t.Fatalf("dir mode %v", di.Mode().Perm())
		}
		if _, err := os.Stat(filepath.Join(dir, "secrets")); err == nil {
			t.Fatal("no sealed file without a portal")
		}
	})

	t.Run("a plain file is sealed once the portal answers", func(t *testing.T) {
		dir := flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no such interface") })
		if err := Set("nexus", "key-123"); err != nil {
			t.Fatal(err)
		}
		portalSecret, keyDone = fixedPortal("master"), false
		if v, err := Get("nexus"); err != nil || v != "key-123" {
			t.Fatalf("got %q, %v", v, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "secrets.json")); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("the plain file is still there: %v", err)
		}
		raw, err := readFile(filepath.Join(dir, "secrets"))
		if err != nil || bytes.Contains(raw, []byte("key-123")) {
			t.Fatalf("the moved secrets must be sealed: %v", err)
		}
	})

	t.Run("no portal uses the host keyring and writes no file", func(t *testing.T) {
		dir := flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no portal") })
		ring := map[string]string{}
		setHostRing(ring)
		if err := Set("nexus", "key-1"); err != nil {
			t.Fatal(err)
		}
		if v, err := Get("nexus"); err != nil || v != "key-1" || ring["nexus"] != "key-1" {
			t.Fatalf("got %q, %v, ring %v", v, err, ring)
		}
		if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("no store folder expected: %v", err)
		}
		if Plaintext() {
			t.Fatal("the keyring is not plain text")
		}
		if err := Delete("nexus"); err != nil || len(ring) != 0 {
			t.Fatalf("delete: %v %v", err, ring)
		}
	})

	t.Run("neither portal nor keyring is plain text", func(t *testing.T) {
		dir := flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no portal") })
		if err := Set("itch", "tok"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "secrets.json")); err != nil || !Plaintext() {
			t.Fatalf("plain file: %v, Plaintext %v", err, Plaintext())
		}
	})

	t.Run("a plain file moves to the keyring once it answers", func(t *testing.T) {
		dir := flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no portal") })
		if err := Set("nexus", "key-1"); err != nil {
			t.Fatal(err)
		}
		ring := map[string]string{}
		setHostRing(ring)
		if v, err := Get("nexus"); err != nil || v != "key-1" || ring["nexus"] != "key-1" {
			t.Fatalf("got %q, %v, ring %v", v, err, ring)
		}
		if _, err := os.Stat(filepath.Join(dir, "secrets.json")); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("the plain file is still there: %v", err)
		}
	})

	t.Run("wrong key errors and leaves the file alone", func(t *testing.T) {
		dir := flatpakEnv(t, fixedPortal("master"))
		if err := Set("nexus", "v"); err != nil {
			t.Fatal(err)
		}
		before, _ := readFile(filepath.Join(dir, "secrets"))
		portalSecret, keyDone = fixedPortal("other"), false
		if _, err := Get("nexus"); err == nil || errors.Is(err, ErrNotFound) {
			t.Fatalf("want a decrypt error, got %v", err)
		}
		if err := Set("x", "y"); err == nil {
			t.Fatal("Set must not overwrite an undecryptable file")
		}
		portalSecret, keyDone = func() ([]byte, error) { return nil, errors.New("gone") }, false
		if err := Set("x", "y"); err == nil {
			t.Fatal("a sealed file without the portal must not be shadowed by a plain one")
		}
		after, _ := readFile(filepath.Join(dir, "secrets"))
		if !bytes.Equal(before, after) {
			t.Fatal("file changed")
		}
	})
}

func TestLivePortal(t *testing.T) {
	if os.Getenv("MORTAR_LIVE_PORTAL") != "1" {
		t.Skip("set MORTAR_LIVE_PORTAL=1 to call the real Secret portal")
	}
	a, err := portal.RetrieveSecret()
	if err != nil || len(a) == 0 {
		t.Fatalf("first call: %q, %v", a, err)
	}
	b, err := portal.RetrieveSecret()
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("second call must return the same secret: %v", err)
	}
}

func TestFlatpakStoreTightensAnExistingFolderAndRefusesALink(t *testing.T) {
	dir := flatpakEnv(t, fixedPortal("master"))
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := Set("nexus", "k"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("folder mode = %v, %v", info.Mode().Perm(), err)
	}

	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := flatpakEnv(t, fixedPortal("master"))
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := Set("nexus", "k"); err == nil {
		t.Fatal("a symlinked secrets folder was used")
	}
}
