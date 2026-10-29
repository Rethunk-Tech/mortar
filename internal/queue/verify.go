package queue

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const packageHashFile = "package-hashes.json"

// verifyDigest checks path against a "sha256:<hex>" or "sha512:<hex>" digest; an empty digest (GitHub did not
// publish one) passes, and so does CurseForge's "sha1:<hex>", which only names the file for update checks.
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
	got, err := fsx.HashFile(path, sha512.New())
	if err != nil {
		return err
	}
	if got != want {
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
	got, err := fsx.HashFile(path, sha256.New())
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
	if s.hashes == nil {
		s.hashes = map[string]string{}
		if _, err := datadir.ReadJSON(file, &s.hashes); err != nil || s.hashes == nil {
			s.hashes = map[string]string{}
		}
	}
	key := strings.ToLower(it.Package) + "@" + it.Version
	if want, ok := s.hashes[key]; ok {
		return compareSHA256(path, want)
	}
	got, err := fsx.HashFile(path, sha256.New())
	if err != nil {
		return err
	}
	s.hashes[key] = got
	if err := datadir.WriteJSON(file, s.hashes); err != nil {
		delete(s.hashes, key)
		return err
	}
	return nil
}
