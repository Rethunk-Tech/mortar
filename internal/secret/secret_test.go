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
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	portalSecret, storeDir = portal, func() (string, error) { return dir, nil }
	setHostRing(nil)
	t.Cleanup(func() {
		portalSecret, storeDir = prevPortal, prevDir
		keyringGet, keyringSet, keyringDelete = prevGet, prevSet, prevDel
		keyDone, hostDone = false, false
		hostChecked = map[string]bool{}
	})
	return dir
}

// setHostRing stubs the host Secret Service with ring, or as unreachable when ring is nil, and re-probes it.
func setHostRing(ring map[string]string) {
	hostChecked = map[string]bool{}
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

func TestStoreDirNeedsAHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("XDG_DATA_HOME", "")
	if d, err := storeDir(); err == nil {
		t.Fatalf("storeDir with no home = %q, want an error", d)
	}
}

func TestSetAndDeleteDiscardAReadStillWaiting(t *testing.T) {
	prev := getTimeout
	getTimeout = 20 * time.Millisecond
	t.Cleanup(func() { getTimeout = prev })
	for _, del := range []bool{false, true} {
		ring := map[string]string{"k": "old"}
		release := make(chan struct{})
		read := make(chan struct{})
		keyringGet = func(_, name string) (string, error) {
			v := ring[name]
			close(read)
			<-release
			return v, nil
		}
		keyringSet = func(_, name, v string) error { ring[name] = v; return nil }
		keyringDelete = func(_, name string) error { delete(ring, name); return nil }
		fired := make(chan struct{}, 1)
		OnUnlocked = func() { fired <- struct{}{} }
		t.Cleanup(func() {
			keyringGet = func(_, name string) (string, error) { return "", errors.New("unset " + name) }
			OnUnlocked = func() {}
		})
		if _, err := Get("k"); !errors.Is(err, ErrWaitingForUnlock) {
			t.Fatal(err)
		}
		<-read
		if del {
			_ = Delete("k")
		} else {
			_ = Set("k", "new")
		}
		close(release)
		select {
		case <-fired:
		case <-time.After(5 * time.Second):
			t.Fatalf("delete=%v: the caller told to wait never heard back", del)
		}
		keyringGet = func(_, name string) (string, error) {
			if v, ok := ring[name]; ok {
				return v, nil
			}
			return "", ErrNotFound
		}
		v, err := Get("k")
		if del && !errors.Is(err, ErrNotFound) || !del && v != "new" {
			t.Fatalf("delete=%v: got %q, %v after the stale read finished", del, v, err)
		}
	}
}

func TestDismissedUnlockIsNotNoKeyring(t *testing.T) {
	dir := flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no portal") })
	dismissed := errors.New("failed to unlock correct collection '/org/freedesktop/secrets/aliases/default'")
	keyringGet = func(string, string) (string, error) { return "", dismissed }
	if err := Set("itch", "tok"); err == nil {
		t.Fatal("a dismissed unlock prompt must not fall back to the plain file")
	}
	if _, err := os.Stat(filepath.Join(dir, "secrets.json")); err == nil {
		t.Fatal("secret written in plain text")
	}
	ring := map[string]string{}
	setHostRing(ring)
	if err := Set("itch", "tok"); err != nil || ring["itch"] != "tok" {
		t.Fatalf("the next call must retry the keyring: %v %v", err, ring)
	}
}

func TestPlaintextDoesNotWaitOnTheStore(t *testing.T) {
	flatpakEnv(t, fixedPortal("master"))
	storeMu.Lock()
	defer storeMu.Unlock()
	if Plaintext() {
		t.Fatal("a busy store is not reported as plain text")
	}
}

// setHostRingBySvc stubs the host Secret Service keyed by service and name.
func setHostRingBySvc(ring map[[2]string]string) {
	hostChecked = map[string]bool{}
	hostDone, keyDone = false, false
	keyringGet = func(svc, name string) (string, error) {
		if v, ok := ring[[2]string{svc, name}]; ok {
			return v, nil
		}
		return "", ErrNotFound
	}
	keyringSet = func(svc, name, v string) error { ring[[2]string{svc, name}] = v; return nil }
	keyringDelete = func(svc, name string) error { delete(ring, [2]string{svc, name}); return nil }
}

func TestHostKeyringSharedItemNaming(t *testing.T) {
	newEnv := func(t *testing.T) map[[2]string]string {
		flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no portal") })
		ring := map[[2]string]string{}
		setHostRingBySvc(ring)
		return ring
	}
	legacy := [2]string{"mortar", "thunderstore.team"}
	shared := [2]string{"thunderstore", "team"}

	t.Run("set writes the host-visible name and drops the old one", func(t *testing.T) {
		ring := newEnv(t)
		ring[legacy] = "old"
		if err := SetIn("thunderstore", "team", "tok"); err != nil {
			t.Fatal(err)
		}
		if ring[shared] != "tok" {
			t.Fatalf("new name not written: %v", ring)
		}
		if _, ok := ring[legacy]; ok {
			t.Fatalf("old name lingers: %v", ring)
		}
	})

	t.Run("get falls back to the old name", func(t *testing.T) {
		ring := newEnv(t)
		ring[legacy], ring[shared] = "old", "new"
		if v, err := GetIn("thunderstore", "team"); err != nil || v != "new" {
			t.Fatalf("got %q, %v", v, err)
		}
		delete(ring, shared)
		if v, err := GetIn("thunderstore", "team"); err != nil || v != "old" {
			t.Fatalf("got %q, %v", v, err)
		}
	})

	t.Run("delete removes both names", func(t *testing.T) {
		ring := newEnv(t)
		ring[legacy], ring[shared] = "old", "tok"
		if err := DeleteIn("thunderstore", "team"); err != nil {
			t.Fatal(err)
		}
		if len(ring) != 0 {
			t.Fatalf("left behind: %v", ring)
		}
	})

	t.Run("Mortar's own items keep their name", func(t *testing.T) {
		ring := newEnv(t)
		if err := Set("nexus", "k"); err != nil {
			t.Fatal(err)
		}
		if ring[[2]string{"mortar", "nexus"}] != "k" || len(ring) != 1 {
			t.Fatalf("got %v", ring)
		}
	})
}

func TestDismissedUnlockIsAnErrorKind(t *testing.T) {
	err := explain(errors.New("failed to unlock correct collection '/x'"))
	var ue *usererr.Error
	if !errors.As(err, &ue) || ue.Kind() != usererr.Unknown || !strings.Contains(err.Error(), "The keyring is locked. Unlock it, then try again.") {
		t.Fatalf("got %v", err)
	}
}

func TestHostItemsAdoptedWhenPortalAppears(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no portal") })
		ring := map[[2]string]string{}
		setHostRingBySvc(ring)
		host := [2]string{"thunderstore", "team"}
		if legacy {
			host = [2]string{"mortar", "thunderstore.team"}
		}
		ring[host] = "tok"
		portalSecret = fixedPortal("portal-secret")
		keyDone, hostDone = false, false
		if v, err := GetIn("thunderstore", "team"); err != nil || v != "tok" {
			t.Fatalf("legacy=%v: got %q, %v", legacy, v, err)
		}
		if len(ring) != 0 {
			t.Fatalf("legacy=%v: host copy left behind: %v", legacy, ring)
		}
		if v, err := GetIn("thunderstore", "team"); err != nil || v != "tok" {
			t.Fatalf("legacy=%v: sealed copy missing: %q, %v", legacy, v, err)
		}
		ring[host] = "tok"
		hostChecked = map[string]bool{}
		if err := DeleteIn("thunderstore", "team"); err != nil || len(ring) != 0 {
			t.Fatalf("legacy=%v: delete left %v, %v", legacy, ring, err)
		}
	}
}

func TestPlainMigrationUsesSharedNaming(t *testing.T) {
	dir := flatpakEnv(t, func() ([]byte, error) { return nil, errors.New("no portal") })
	if err := save(filepath.Join(dir, plainFile), nil, map[string]string{"thunderstore.team.x": "t", "nexus": "k"}); err != nil {
		t.Fatal(err)
	}
	ring := map[[2]string]string{}
	setHostRingBySvc(ring)
	if v, err := GetIn("thunderstore", "team.x"); err != nil || v != "t" {
		t.Fatalf("got %q, %v", v, err)
	}
	if ring[[2]string{"thunderstore", "team.x"}] != "t" || ring[[2]string{"mortar", "nexus"}] != "k" || len(ring) != 2 {
		t.Fatalf("ring %v", ring)
	}
}

func TestSealedHostLookupOncePerRun(t *testing.T) {
	// The probe in location() must see a Secret Service, so only the item lookups are locked.
	setup := func(t *testing.T, ring map[string]string, locked bool) (reads *int) {
		flatpakEnv(t, fixedPortal("master"))
		setHostRing(ring)
		n := 0
		get, del := keyringGet, keyringDelete
		keyringGet = func(svc, name string) (string, error) {
			if name == hostProbeName {
				return get(svc, name)
			}
			n++
			if locked {
				return "", errors.New("failed to unlock correct collection '/x'")
			}
			return get(svc, name)
		}
		keyringDelete = func(svc, name string) error {
			if locked {
				return errors.New("failed to unlock correct collection '/x'")
			}
			return del(svc, name)
		}
		return &n
	}

	t.Run("locked host is not-found and read once", func(t *testing.T) {
		reads := setup(t, map[string]string{"k": "v"}, true)
		for range 2 {
			if _, err := Get("k"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("got %v", err)
			}
		}
		if *reads != 1 {
			t.Fatalf("host read %d times", *reads)
		}
	})

	t.Run("unlocked host is adopted", func(t *testing.T) {
		ring := map[string]string{"k": "v"}
		setup(t, ring, false)
		if v, err := Get("k"); err != nil || v != "v" {
			t.Fatalf("got %q, %v", v, err)
		}
		if _, ok := ring["k"]; ok {
			t.Fatal("host copy not removed")
		}
		if v, err := Get("k"); err != nil || v != "v" {
			t.Fatalf("sealed copy lost: %q, %v", v, err)
		}
	})

	t.Run("delete with a locked host succeeds", func(t *testing.T) {
		setup(t, map[string]string{}, true)
		if err := Set("k", "v"); err != nil {
			t.Fatal(err)
		}
		if err := Delete("k"); err != nil {
			t.Fatal(err)
		}
	})
}
