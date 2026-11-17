package sharehost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestStoreRoundTripAndRejectsTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st := Store{Dir: dir}
	rec := Record{Code: "AbCdEfGh12", Token: testToken, Version: 1}
	if err := st.Put("stardew", "0123456789abcdef", rec); err != nil {
		t.Fatal(err)
	}
	got, found, err := st.Get("stardew", "0123456789abcdef")
	if err != nil || !found || got != rec {
		t.Fatalf("get = %+v found=%v err=%v", got, found, err)
	}
	if _, found, err = st.Get("stardew", "fedcba9876543210"); err != nil || found {
		t.Fatalf("missing = found=%v err=%v", found, err)
	}
	if _, err := st.file("stardew", "../etc"); err == nil {
		t.Fatal("traversal was accepted")
	}
	profileJSON := filepath.Join(dir, "profiles", "stardew", "0123456789abcdef", "profile.json")
	if _, err := os.Stat(profileJSON); !os.IsNotExist(err) {
		t.Fatalf("wrote profile.json: %v", err)
	}
	raw := testfs.Read(t, filepath.Join(dir, "secrets", "shares", "stardew", "0123456789abcdef.json"))
	if !strings.Contains(raw, testToken) {
		t.Fatal("token was not stored in the secrets file")
	}
}
