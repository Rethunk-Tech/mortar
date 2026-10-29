package secret

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fixtureStore copies a testdata file into a fresh store folder under name.
func fixtureStore(t *testing.T, portal func() ([]byte, error), fixture, name string) string {
	t.Helper()
	dir := flatpakEnv(t, portal)
	b, err := readFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := root.WriteFile(name, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStoreConstants(t *testing.T) {
	if service != "mortar" || hkdfInfo != "mortar secrets v1" || sealedFile != "secrets" || plainFile != "secrets.json" {
		t.Fatalf("constants changed: %q %q %q %q", service, hkdfInfo, sealedFile, plainFile)
	}
	dir, err := storeDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "mortar-secrets" {
		t.Fatalf("store folder %q", dir)
	}
	if len(sharedNamespaces) != 1 || sharedNamespaces[0] != "thunderstore" {
		t.Fatalf("shared namespaces %v", sharedNamespaces)
	}
}

// sealed-0.3 was written by the 0.3 seal under portal secret "fixture-portal-secret" (genfixture_test.go).
func TestSealedFixtureOpens(t *testing.T) {
	fixtureStore(t, fixedPortal("fixture-portal-secret"), "sealed-0.3", sealedFile)
	if v, err := Get("nexus"); err != nil || v != "k" {
		t.Fatalf("nexus: %q, %v", v, err)
	}
	if v, err := GetIn("thunderstore", "team"); err != nil || v != "t" {
		t.Fatalf("thunderstore.team: %q, %v", v, err)
	}
}

func TestPlainFixtureMigrates(t *testing.T) {
	noPortal := func() ([]byte, error) { return nil, errors.New("no portal") }
	t.Run("sealed", func(t *testing.T) {
		dir := fixtureStore(t, noPortal, "secrets-0.3.json", plainFile)
		portalSecret = fixedPortal("fixture-portal-secret")
		keyDone = false
		if v, err := Get("nexus"); err != nil || v != "k" {
			t.Fatalf("nexus: %q, %v", v, err)
		}
		if v, err := GetIn("thunderstore", "team"); err != nil || v != "t" {
			t.Fatalf("thunderstore.team: %q, %v", v, err)
		}
		if _, err := os.Stat(filepath.Join(dir, plainFile)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("plain file left behind: %v", err)
		}
	})
	t.Run("host keyring", func(t *testing.T) {
		fixtureStore(t, noPortal, "secrets-0.3.json", plainFile)
		ring := map[[2]string]string{}
		setHostRingBySvc(ring)
		if v, err := Get("nexus"); err != nil || v != "k" {
			t.Fatalf("nexus: %q, %v", v, err)
		}
		if ring[[2]string{"mortar", "nexus"}] != "k" || ring[[2]string{"thunderstore", "team"}] != "t" || len(ring) != 2 {
			t.Fatalf("ring %v", ring)
		}
	})
}
