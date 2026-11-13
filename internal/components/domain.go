package components

import "crypto/ed25519"

// List names used in the v2 signature domain. A signature over one list must
// not verify as another.
const (
	ListCatalog      = "catalog"
	ListKnownBroken  = "known-broken"
	ListSuppressions = "suppressions"
)

const (
	sigFormV2     = "v2"
	sigFormLegacy = "legacy"
)

func domainSeparated(list string, payload []byte) []byte {
	prefix := "mortar-" + list + "-v1\n"
	out := make([]byte, len(prefix)+len(payload))
	copy(out, prefix)
	copy(out[len(prefix):], payload)
	return out
}

// SignV2 returns an Ed25519 signature over "mortar-<list>-v1\n" + payload.
func SignV2(private ed25519.PrivateKey, list string, payload []byte) []byte {
	return ed25519.Sign(private, domainSeparated(list, payload))
}

// VerifyV2 checks an Ed25519 signature over "mortar-<list>-v1\n" + payload.
func VerifyV2(payload, signature, public []byte, list string) error {
	return Verify(domainSeparated(list, payload), signature, public)
}

func verifyCached(payload, signature, public []byte, list, form string) error {
	if form == sigFormV2 {
		return VerifyV2(payload, signature, public, list)
	}
	return Verify(payload, signature, public)
}
