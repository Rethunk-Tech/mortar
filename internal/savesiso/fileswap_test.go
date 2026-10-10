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
