package queue

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const packageHashFile = "package-hashes.json"

// verifyDigest checks path against a "sha256:<hex>" or "sha512:<hex>" digest; an empty digest (GitHub did not
// publish one) passes.
func verifyDigest(path, digest string) error {
	digest = strings.ToLower(strings.TrimSpace(digest))
	if want, ok := strings.CutPrefix(digest, "sha512:"); ok {
		return compareSHA512(path, want)
	}
	want, ok := strings.CutPrefix(digest, "sha256:")
	if !ok {
		return nil
	}
	return compareSHA256(path, want)
}

func compareSHA512(path, want string) error {
	f, err := fsx.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	h := sha512.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return usererr.New(usererr.Damaged, "download is damaged or changed: SHA-512 does not match the one the site published")
	}
	return nil
}

// checkPackage verifies a downloaded package: Modrinth publishes a digest to check, Thunderstore does not and is
// held to its first download.
func (s *Service) checkPackage(it Item, path string) error {
	if it.Source != "" {
		return verifyDigest(path, it.Digest)
	}
	return s.checkPackageHash(it, path)
}

func compareSHA256(path, want string) error {
	got, err := fsx.SHA256(path)
	if err != nil {
		return err
	}
	if got != want {
		return usererr.New(usererr.Damaged, fmt.Sprintf("download is damaged or changed: SHA-256 is %s, expected %s", got, want))
	}
	return nil
}

// checkPackageHash holds a Thunderstore package version to the SHA-256 of its first download, since the index
// publishes no hash: a later download that differs fails instead of installing.
func (s *Service) checkPackageHash(it Item, path string) error {
	s.hashMu.Lock()
	defer s.hashMu.Unlock()
	file := filepath.Join(s.d.Dir, packageHashFile)
	seen := map[string]string{}
	if _, err := datadir.ReadJSON(file, &seen); err != nil {
		seen = map[string]string{}
	}
	key := strings.ToLower(it.Package) + "@" + it.Version
	if want, ok := seen[key]; ok {
		return compareSHA256(path, want)
	}
	got, err := fsx.SHA256(path)
	if err != nil {
		return err
	}
	seen[key] = got
	return datadir.WriteJSON(file, seen)
}
