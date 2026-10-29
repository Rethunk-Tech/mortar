//go:build genfixture

package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"os"
	"testing"
)

// TestGenSealedFixture rewrites testdata/sealed-0.3 in the sealed-file format (nonce, then AES-256-GCM):
// go test -tags genfixture -run TestGenSealedFixture ./internal/secret
func TestGenSealedFixture(t *testing.T) {
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
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	b := g.Seal(nonce, nonce, []byte(`{"nexus":"k","thunderstore.team":"t"}`), nil)
	if err := os.WriteFile("testdata/sealed-0.3", b, 0o600); err != nil {
		t.Fatal(err)
	}
}
