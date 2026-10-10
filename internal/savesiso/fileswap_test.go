package savesiso

import (
	"os"
	"path/filepath"
	"testing"
)

func fileEnv(t *testing.T) (target, prof, journal string) {
	root := t.TempDir()
	return filepath.Join(root, "g", "Options.ini"), filepath.Join(root, "p", "Options.ini"), filepath.Join(root, "j")
}

func TestFileSwapKeepsPlayersFileAndProfileCopyApart(t *testing.T) {
	target, prof, journal := fileEnv(t)
	write(t, target, "player")
	if err := SeedFile(target, prof); err != nil || read(t, prof) != "player" {
		t.Fatalf("seed: %v", err)
	}
	write(t, prof, "profile")
	if err := SeedFile(target, prof); err != nil || read(t, prof) != "profile" {
		t.Fatalf("a seed must not replace an existing profile copy: %v", err)
	}
	m, err := ApplyFile(journal, target, prof)
	if err != nil {
		t.Fatal(err)
	}
	if read(t, target) != "profile" {
		t.Fatal("the game must see the profile's file")
	}
	write(t, target, "played")
	if err := PurgeFile(m); err != nil {
		t.Fatal(err)
	}
	if read(t, target) != "player" || read(t, prof) != "played" || exists(journal) || exists(target+heldSuffix) {
		t.Fatal("the player's file returns untouched and the profile keeps what was played")
	}
}

func TestFileSwapWithNoPlayersFileLeavesNoneBehind(t *testing.T) {
	target, prof, journal := fileEnv(t)
	m, err := ApplyFile(journal, target, prof)
	if err != nil {
		t.Fatal(err)
	}
	write(t, target, "created")
	if err := PurgeFile(m); err != nil {
		t.Fatal(err)
	}
	if exists(target) || read(t, prof) != "created" {
		t.Fatal("a file the game made belongs to the profile")
	}
}

func TestFileRecoverAfterACrashRestoresPlayersFile(t *testing.T) {
	target, prof, journal := fileEnv(t)
	write(t, target, "player")
	write(t, prof, "profile")
	if _, err := ApplyFile(journal, target, prof); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyFile(journal, target, prof); err == nil {
		t.Fatal("a second swap over an unrecovered one must be refused")
	}
	write(t, target, "played")
	if err := RecoverFile(journal, func() bool { return true }); err != nil || read(t, target) != "played" {
		t.Fatalf("a running game's swap is left alone: %v", err)
	}
	if err := RecoverFile(journal, nil); err != nil {
		t.Fatal(err)
	}
	if read(t, target) != "player" || read(t, prof) != "played" || HasFileJournal(journal) {
		t.Fatal("recovery must return the player's file")
	}
}

// A kill after the game's file was written back and removed, before the player's file returned.
func TestFileRecoverAfterKillMidPurge(t *testing.T) {
	target, prof, journal := fileEnv(t)
	write(t, target, "player")
	write(t, prof, "profile")
	if _, err := ApplyFile(journal, target, prof); err != nil {
		t.Fatal(err)
	}
	write(t, prof, "played")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := RecoverFile(journal, nil); err != nil {
		t.Fatal(err)
	}
	if read(t, target) != "player" || read(t, prof) != "played" || HasFileJournal(journal) {
		t.Fatal("the held file must return and the profile copy stay")
	}
}

// A crash after the record is written and before the player's file is renamed leaves the player's own file at its
// path; recovery must not mistake it for the profile's copy, overwrite the profile's file with it and delete it.
func TestRecoverAfterACrashBeforeThePlayersFileIsSetAside(t *testing.T) {
	target, prof, journal := fileEnv(t)
	write(t, target, "player")
	write(t, prof, "profile")
	if err := persistFile(FileManifest{Journal: journal, Target: target, Profile: prof, Held: target + heldSuffix}); err != nil {
		t.Fatal(err)
	}
	if err := RecoverFile(journal, nil); err != nil {
		t.Fatal(err)
	}
	if read(t, target) != "player" || read(t, prof) != "profile" {
		t.Fatalf("player's file = %q (want player), profile copy = %q (want profile)", read(t, target), read(t, prof))
	}
}

// Every pair of adjacent steps of ApplyFile and PurgeFile, killed between: the on-disk state each leaves is rebuilt
// by hand, and recovery must return the player's file byte for byte.
func TestFileRecoverFromEveryKillPoint(t *testing.T) {
	held := func(target string) string { return target + heldSuffix }
	for _, c := range []struct {
		name  string
		setup func(t *testing.T, target, prof string)
		// profile is the profile copy recovery must leave.
		profile string
	}{
		{"apply: record written, nothing moved", func(t *testing.T, target, prof string) {
			write(t, target, "player")
		}, "profile"},
		{"apply: player's file set aside, nothing placed", func(t *testing.T, target, prof string) {
			write(t, held(target), "player")
		}, "profile"},
		{"apply: profile copy placed", func(t *testing.T, target, prof string) {
			write(t, held(target), "player")
			write(t, target, "profile")
		}, "profile"},
		{"apply: game wrote while running", func(t *testing.T, target, prof string) {
			write(t, held(target), "player")
			write(t, target, "played")
		}, "played"},
		{"purge: written back, game's file still there", func(t *testing.T, target, prof string) {
			write(t, held(target), "player")
			write(t, target, "played")
			write(t, prof, "played")
		}, "played"},
		{"purge: game's file removed, player's still aside", func(t *testing.T, target, prof string) {
			write(t, held(target), "player")
			write(t, prof, "played")
		}, "played"},
		{"purge: player's file returned, record still there", func(t *testing.T, target, prof string) {
			write(t, target, "player")
			write(t, prof, "played")
		}, "played"},
	} {
		target, prof, journal := fileEnv(t)
		write(t, prof, "profile")
		c.setup(t, target, prof)
		if err := persistFile(FileManifest{Journal: journal, Target: target, Profile: prof, Held: held(target)}); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if err := RecoverFile(journal, nil); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		}
		if read(t, target) != "player" || read(t, prof) != c.profile || HasFileJournal(journal) || exists(held(target)) {
			t.Errorf("%s: player's file %q, profile copy %q", c.name, read(t, target), read(t, prof))
		}
	}
}

func TestSeedFileTakesTheHeldFileWhileAnotherInstallSwapsItIn(t *testing.T) {
	target, prof, _ := fileEnv(t)
	write(t, target, "other install's profile copy")
	write(t, target+heldSuffix, "player")
	if err := SeedFile(target, prof); err != nil || read(t, prof) != "player" {
		t.Fatalf("seed = %q (%v), want the player's file", read(t, prof), err)
	}
}
