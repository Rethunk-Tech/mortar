package components

import (
	"crypto/ed25519"
	"testing"
)

func TestCatalogSignatureDoesNotVerifyAsSuppressions(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("shared-payload")
	sig := Sign(ListCatalog, payload, priv)
	if Verify(ListSuppressions, payload, sig, pub) == nil {
		t.Fatal("catalog signature verified as suppressions")
	}
	if err := Verify(ListCatalog, payload, sig, pub); err != nil {
		t.Fatal(err)
	}
}
