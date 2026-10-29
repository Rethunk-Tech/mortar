//go:build genfixture

package secret

import (
	"crypto/hkdf"
	"crypto/sha256"
	"os"
	"testing"
)

// TestGenSealedFixture rewrites testdata/sealed-0.3 with the seal in this tree:
// go test -tags genfixture -run TestGenSealedFixture ./internal/secret
func TestGenSealedFixture(t *testing.T) {
	key, err := hkdf.Key(sha256.New, []byte("fixture-portal-secret"), nil, hkdfInfo, 32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := seal(key, []byte(`{"nexus":"k","thunderstore.team":"t"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("testdata/sealed-0.3", b, 0o600); err != nil {
		t.Fatal(err)
	}
}
