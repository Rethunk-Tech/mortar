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

	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

const protocolVersion = "3"

type nonceRecord struct {
	peer    string
	expires time.Time
}

type helloResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Nonce   string `json:"nonce"`
}

type shareResponse struct {
	Paired        bool     `json:"paired"`
	TransferToken string   `json:"transferToken,omitempty"`
	EntryKeys     []string `json:"entryKeys,omitempty"`
	// Proof shows the sender that the receiver holds the pairing key, so a computer posing as a paired one cannot
	// hand out a transfer token and pull files.
	Proof string `json:"proof,omitempty"`
}

func hmacProof(key []byte, nonce, payload string) string {
	payloadHash := sha256.Sum256([]byte(payload))
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(nonce))
	_, _ = mac.Write(payloadHash[:])
	return hex.EncodeToString(mac.Sum(nil))
}

func proofMatches(key []byte, nonce, payload, proof string) bool {
	if len(key) == 0 || nonce == "" || proof == "" {
		return false
	}
	return hmac.Equal([]byte(hmacProof(key, nonce, payload)), []byte(proof))
}

// responseProof binds a receiver's transfer token and key list to the share it answers.
func responseProof(key []byte, nonce, token string, keys []string) string {
	return hmacProof(key, nonce, "response|"+token+"|"+strings.Join(keys, ","))
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// transferItem is one store entry a paired receiver copies from the sender, with what the store records about
// its source.
type transferItem struct {
	Key     string
	Source  string
	Package string
	Version string
}

// transferItems lists the store entries a share's files live under, whatever their source. A package without a
// version is the newest at the receiver, so it has no key to copy.
func transferItems(shared share.Shared) []transferItem {
	items := make([]transferItem, 0, len(shared.Entries))
	for _, ref := range shared.Entries {
		var item transferItem
		switch {
		case ref.GitHub != "":
			repo, tag, asset := ref.GitHubParts()
			owner, name, _ := strings.Cut(repo, "/")
			item = transferItem{Key: github.Key(owner, name, tag, asset), Source: profile.KindGitHub, Package: repo, Version: tag}
		case ref.Package != "":
			if ref.Version == "" {
				continue
			}
			item = transferItem{Key: store.PackageKey(ref.Package, ref.Version), Source: profile.KindThunderstore, Package: ref.Package, Version: ref.Version}
		default:
			item = transferItem{Key: store.NexusKey(ref.ModID, ref.FileID)}
		}
		if !slices.ContainsFunc(items, func(have transferItem) bool { return have.Key == item.Key }) {
			items = append(items, item)
		}
	}
	return items
}
