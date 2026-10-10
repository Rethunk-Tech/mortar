package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

// A created file over the per-file cap stays shared and is not copied into the profile.
func TestACreatedFileOverTheCapStaysSharedAndIsNotCopied(t *testing.T) {
	rr := newRootRig(t)
	m := rr.apply()
	big := filepath.Join(rr.shared, "mc", "world.dat")
	write(t, big, "")
	if err := os.Truncate(big, maxAdoptFile+1); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(rr.shared, "mc", "small.dat"), "small")
	rr.purge(m)
	if st, err := os.Stat(big); err != nil || st.Size() != maxAdoptFile+1 {
		t.Fatalf("the over-cap file left the shared folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rr.profile, "mc", "world.dat")); err == nil {
		t.Fatal("the over-cap file was copied into the profile")
	}
	if read(filepath.Join(rr.profile, "mc", "small.dat")) != "small" {
		t.Fatal("a small created file beside it was not adopted")
	}
}
