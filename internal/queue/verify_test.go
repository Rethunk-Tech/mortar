package queue

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.zip")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestVerifyDigest(t *testing.T) {
	sum := sha256.Sum256([]byte("abc"))
	p := writeFile(t, "abc")
	if err := verifyDigest(p, "sha256:"+hex.EncodeToString(sum[:])); err != nil {
		t.Errorf("matching digest: %v", err)
	}
	if err := verifyDigest(p, "sha256:00"); err == nil {
		t.Error("a wrong digest passed")
	}
	if err := verifyDigest(p, ""); err != nil {
		t.Errorf("no digest: %v", err)
	}
}

func TestPackageHashHoldsLaterCopiesToTheFirst(t *testing.T) {
	s := &Service{d: Deps{Dir: t.TempDir()}}
	it := Item{Package: "Me-Mod", Version: "1.0.0"}
	if err := s.checkPackageHash(it, writeFile(t, "one")); err != nil {
		t.Fatal(err)
	}
	if err := s.checkPackageHash(it, writeFile(t, "one")); err != nil {
		t.Errorf("same bytes: %v", err)
	}
	if err := s.checkPackageHash(it, writeFile(t, "two")); err == nil {
		t.Error("changed bytes passed")
	}
	it.Version = "1.0.1"
	if err := s.checkPackageHash(it, writeFile(t, "two")); err != nil {
		t.Errorf("new version: %v", err)
	}
}
