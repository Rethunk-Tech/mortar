package lan

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

const protocolVersion = "2"

type nonceRecord struct {
	peer    string
	expires time.Time
}

type helloResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Nonce   string `json:"nonce"`
}

type shareResponse struct {
	SameAccount   bool     `json:"sameAccount"`
	TransferToken string   `json:"transferToken,omitempty"`
	EntryKeys     []string `json:"entryKeys,omitempty"`
}

func hmacProof(key, nonce, payload string) string {
	payloadHash := sha256.Sum256([]byte(payload))
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(nonce))
	_, _ = mac.Write(payloadHash[:])
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *Service) nexusKey() string {
	if s.deps.NexusKey == nil {
		return ""
	}
	key, err := s.deps.NexusKey()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(key)
}

func accountMatches(key, nonce, payload, proof string) bool {
	if key == "" || nonce == "" || proof == "" {
		return false
	}
	expected := hmacProof(key, nonce, payload)
	return hmac.Equal([]byte(expected), []byte(proof))
}

func entryKeys(shared share.Shared) []string {
	keys := make([]string, 0, len(shared.Entries))
	for _, ref := range shared.Entries {
		if ref.GitHub != "" || ref.Package != "" {
			continue
		}
		key := store.NexusKey(ref.ModID, ref.FileID)
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys
}
