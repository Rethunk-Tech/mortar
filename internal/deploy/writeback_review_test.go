package deploy

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A file of the player's own inside a folder an entry owns is the player's even when the game rewrites it: only a file
// created during play is adopted, and the player's stays in the shared folder.
func TestAPlayersOwnFileInAnOwnedFolderStaysWhenTheGameRewritesIt(t *testing.T) {
	rr := newRootRig(t)
	own := filepath.Join(rr.shared, "mc", "players_own.cfg")
	write(t, own, "mine")
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(own, past, past); err != nil {
		t.Fatal(err)
	}
	m := rr.apply()
	write(t, own, "mine, rewritten by the game")
	rr.purge(m)
	if got := read(own); got != "mine, rewritten by the game" {
		t.Fatalf("the player's file = %q (adopted into the profile and removed from the shared folder)", got)
	}
}

// A mod that replaces a file it was placed with a link to a file elsewhere must not get that file's bytes copied into
// the profile (where backups and shares carry them), and the target stays as it was.
func TestALinkInPlaceOfAPlacedFileIsNotFollowedIntoTheProfile(t *testing.T) {
	rr := newRootRig(t)
	secret := filepath.Join(rr.root, "outside", "secret.txt")
	write(t, secret, "secret bytes")
	m := rr.apply()
	dst := filepath.Join(rr.shared, "mc", "mc_settings.cfg")
	if err := os.Remove(dst); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, dst); err != nil {
		t.Skip("links are not available")
	}
	rr.purge(m)
	if got := read(filepath.Join(rr.profile, "mc", "mc_settings.cfg")); strings.Contains(got, "secret") {
		t.Fatalf("the profile's copy holds the linked file's bytes: %q", got)
	}
	if read(secret) != "secret bytes" {
		t.Fatal("the link's target changed")
	}
}

// A link a mod leaves in its own folder is not adopted, followed or deleted through.
func TestALinkedFileOrFolderInAnOwnedFolderIsNotAdopted(t *testing.T) {
	rr := newRootRig(t)
	outside := filepath.Join(rr.root, "outside")
	write(t, filepath.Join(outside, "f.txt"), "outside file")
	write(t, filepath.Join(outside, "dir", "g.txt"), "outside dir file")
	m := rr.apply()
	if err := os.Symlink(filepath.Join(outside, "f.txt"), filepath.Join(rr.shared, "mc", "link.txt")); err != nil {
		t.Skip("links are not available")
	}
	if err := os.Symlink(filepath.Join(outside, "dir"), filepath.Join(rr.shared, "mc", "linkdir")); err != nil {
		t.Fatal(err)
	}
	rr.purge(m)
	var adopted []string
	_ = filepath.WalkDir(rr.profile, func(p string, d fs.DirEntry, _ error) error {
		if d != nil && !d.IsDir() && (strings.HasSuffix(p, "link.txt") || strings.HasSuffix(p, "g.txt")) {
			adopted = append(adopted, p)
		}
		return nil
	})
	if len(adopted) > 0 {
		t.Fatalf("adopted through a link: %v", adopted)
	}
	if read(filepath.Join(outside, "f.txt")) != "outside file" || read(filepath.Join(outside, "dir", "g.txt")) != "outside dir file" {
		t.Fatal("a link's target was changed")
	}
}

// No bytes are removed without a copy: when the profile's folder is gone and a changed file displaced the player's,
// the player's file returns, and the changed bytes must still be somewhere on disk.
func TestChangedBytesSurviveWhenTheProfileIsGoneAndTheyDisplacedAPlayersFile(t *testing.T) {
	rr := newRootRig(t)
	write(t, filepath.Join(rr.shared, "top.package"), "the player's own")
	m := rr.apply()
	write(t, filepath.Join(rr.shared, "top.package"), "rewritten top")
	if err := os.RemoveAll(filepath.Dir(rr.profile)); err != nil {
		t.Fatal(err)
	}
	rr.purge(m)
	found := false
	_ = filepath.WalkDir(rr.root, func(p string, d fs.DirEntry, _ error) error {
		if d != nil && d.Type().IsRegular() {
			if b, err := fsx.ReadFile(p); err == nil && string(b) == "rewritten top" {
				found = true
			}
		}
		return nil
	})
	if !found {
		t.Fatal("the changed bytes were removed with no copy anywhere")
	}
}
