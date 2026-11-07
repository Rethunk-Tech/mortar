package lan

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"time"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

const protocolVersion = "6"

type nonceRecord struct {
	peer    string
	expires time.Time
	// challenge, callerID and host are what the caller sent or dialed, kept to derive the session key when the sealed
	// share arrives.
	challenge, callerID, host string
}

type helloResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Nonce   string `json:"nonce"`
	// Proof answers the caller's challenge with the pairing key, so a host merely claiming a paired id is treated as
	// unpaired. An id this computer is not paired with gets a proof under a throwaway key, so the answer never says
	// which ids are paired.
	Proof string `json:"proof,omitempty"`
	// challenge is what this computer sent and host the address it dialed; neither travels back.
	challenge, host string
}

// helloProof binds a /hello answer to the caller's challenge, both ids in a fixed order (caller, then answerer) and
// the address the caller dialed, so an answer relayed from another computer fails.
func helloProof(key []byte, challenge, callerID, answererID, host string) string {
	return hmacProof(key, challenge, "hello|"+callerID+"|"+answererID+"|"+host)
}

type shareResponse struct {
	Paired        bool     `json:"paired"`
	TransferToken string   `json:"transferToken,omitempty"`
	EntryKeys     []string `json:"entryKeys,omitempty"`
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

// entryDigest is a store entry's content hash and the sender's MAC over it.
type entryDigest struct {
	Hash string `json:"hash"`
	MAC  string `json:"mac"`
}

// entryMAC binds an entry's content hash to the game and key under the pair key, so the receiver can tell what the
// sender holds from what an eavesdropper serves on the unauthenticated transfer address.
func entryMAC(key []byte, game, entry, hash string) string {
	return hmacProof(key, "store|"+game+"|"+entry, hash)
}

func randomToken() string {
	raw := make([]byte, 32)
	_, _ = rand.Read(raw) // never fails since Go 1.24
	return base64.RawURLEncoding.EncodeToString(raw)
}

// transferItem is one store entry a paired receiver copies from the sender, with what the store records about
// its source.
type transferItem struct {
	Key string
	// Hash is the content hash the sender vouched for with the pair key; empty when it vouched for none.
	Hash    string
	Source  string
	Package string
	Version string
	// Asset is a GitHub item's release asset name.
	Asset string
}

// transferItems lists the store entries a share's files live under, whatever their source. A package without a
// version is the newest at the receiver, so it has no key to copy.
func transferItems(shared share.Shared) []transferItem {
	items := make([]transferItem, 0, len(shared.Entries))
	for _, ref := range shared.Entries {
		var item transferItem
		switch {
		case ref.Local != "":
			item = transferItem{Key: ref.Local, Source: profile.KindLocal, Package: ref.LocalName}
		case ref.Patreon != "", ref.Itch != "":
			// A link carries the page and no file; the receiver saves the file from the page.
			continue
		case ref.CurseForge != 0:
			project := strconv.Itoa(ref.CurseForge)
			item = transferItem{Key: store.PackageKey(profile.KindCurseForge+":"+project, strconv.Itoa(ref.FileID)), Source: profile.KindCurseForge, Package: project}
		case ref.VSModDB != "":
			item = transferItem{Key: store.PackageKey(profile.KindVSModDB+":"+ref.VSModDB, ref.Version), Source: profile.KindVSModDB, Package: ref.VSModDB, Version: ref.Version}
		case ref.Modio != 0:
			mod := strconv.Itoa(ref.Modio)
			item = transferItem{Key: store.PackageKey(profile.KindModio+":"+mod, strconv.Itoa(ref.FileID)), Source: profile.KindModio, Package: mod}
		case ref.Paradox != "":
			item = transferItem{Key: store.PackageKey(profile.KindParadox+":"+ref.Paradox, ref.Version), Source: profile.KindParadox, Package: ref.Paradox, Version: ref.Version}
		case ref.GitHub != "":
			repo, tag, asset := ref.GitHubParts()
			owner, name, _ := strings.Cut(repo, "/")
			item = transferItem{Key: gh.Key(owner, name, tag, asset), Source: profile.KindGitHub, Package: repo, Version: tag, Asset: asset}
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
