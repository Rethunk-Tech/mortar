package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-keyring"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	zkeyring "github.com/zalando/go-keyring"
)

func TestStoreConstants(t *testing.T) {
	if service != "mortar" || sealInfo != "mortar secrets v1" || folder != "mortar-secrets" {
		t.Fatalf("constants changed: %q %q %q", service, sealInfo, folder)
	}
}

// sealed-0.3 was written by the 0.3 seal under portal secret "fixture-portal-secret" (genfixture_test.go). Opening it
// here with the standard library proves Mortar's sealInfo is the one 0.3 used.
func TestSealedFixtureOpens(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "sealed-0.3"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := hkdf.Key(sha256.New, []byte("fixture-portal-secret"), nil, sealInfo, 32)
	if err != nil {
		t.Fatal(err)
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	g, err := cipher.NewGCM(blk)
	if err != nil {
		t.Fatal(err)
	}
	n := g.NonceSize()
	got, err := g.Open(nil, b[:n], b[n:], nil)
	if err != nil || string(got) != `{"nexus":"k","thunderstore.team":"t"}` {
		t.Fatalf("sealed-0.3 opens to %q, %v", got, err)
	}
}

// With no portal the 0.3 plain file moves into the host keyring, a Thunderstore item back under its own service.
func TestPlainFixtureMigrates(t *testing.T) {
	zkeyring.MockInit()
	dir := t.TempDir()
	t.Setenv("FLATPAK_ID", "tech.rethunk.Mortar")
	t.Setenv("HOME", dir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path="+filepath.Join(dir, "no-bus"))
	b, err := os.ReadFile(filepath.Join("testdata", "secrets-0.3.json"))
	if err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(dir, "data", folder)
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(store)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := root.WriteFile("secrets.json", b, 0o600); err != nil {
		t.Fatal(err)
	}
	if v, err := Get("nexus"); err != nil || v != "k" {
		t.Fatalf("nexus: %q, %v", v, err)
	}
	if v, err := zkeyring.Get("thunderstore", "team"); err != nil || v != "t" {
		t.Fatalf("thunderstore/team: %q, %v", v, err)
	}
	if v, err := zkeyring.Get("mortar", "nexus"); err != nil || v != "k" {
		t.Fatalf("mortar/nexus: %q, %v", v, err)
	}
	if _, err := os.Stat(filepath.Join(store, "secrets.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("plain file left behind: %v", err)
	}
}

func TestMapErr(t *testing.T) {
	if err := mapErr(keyring.ErrPending); !errors.Is(err, ErrWaitingForUnlock) {
		t.Fatalf("pending: %v", err)
	}
	if usererr.KindOf(ErrWaitingForUnlock) != usererr.Locked {
		t.Fatal("ErrWaitingForUnlock lost its kind")
	}
	for err, want := range map[error]string{
		errors.Join(keyring.ErrNoProvider, errors.New("org.freedesktop.DBus.Error.ServiceUnknown")): "No keyring service is running",
		errors.Join(keyring.ErrLocked, errors.New("failed to unlock correct collection")):           "The keyring is locked",
	} {
		got := mapErr(err)
		if !strings.Contains(got.Error(), want) || strings.Contains(got.Error(), "DBus") || usererr.KindOf(got) != usererr.Unknown {
			t.Fatalf("%v maps to %q", err, got)
		}
	}
	other := errors.New("boom")
	if !errors.Is(mapErr(other), other) || mapErr(nil) != nil || !errors.Is(mapErr(ErrNotFound), ErrNotFound) || !errors.Is(mapErr(ErrTooBig), ErrTooBig) {
		t.Fatal("unrelated errors must pass through")
	}
}

// A zalando bump changes how items are stored, so it lands in go-keyring first, where its live tests run.
func TestZalandoMatchesGoKeyring(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("no build info")
	}
	for _, d := range info.Deps {
		if d.Path == "github.com/zalando/go-keyring" {
			if d.Version != keyring.ZalandoVersion {
				t.Fatalf("zalando %s, go-keyring tested %s", d.Version, keyring.ZalandoVersion)
			}
			return
		}
	}
	t.Fatal("zalando missing from build info")
}
