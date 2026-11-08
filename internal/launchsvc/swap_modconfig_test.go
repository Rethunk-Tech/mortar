//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/savesiso"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func TestSwapModConfigSeedsOnceAndRecoversACrashJournal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	svc, pid, _, inst := serverEnv(t)
	svc.procDir = t.TempDir()
	shared, err := game.PathFor(svc.home, svc.settings.Get(), vsGame, "", profile.ModConfigRole)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteAt(t, filepath.Join(shared, "mod.json"), []byte("first"))
	dir, err := svc.profiles.ProfileDir(vsGame, pid)
	if err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(dir, profile.ModConfigFolder)
	dep := &deployment{}
	if err := svc.swapModConfig(t.Context(), vsGame, inst, pid, "", dep); err != nil {
		t.Fatal(err)
	}
	if dep.modConfig == nil || dep.finish == nil {
		t.Fatalf("the swap is not tracked for undoing: %+v", dep)
	}
	if b, err := safefs.ReadFile(filepath.Join(own, "mod.json")); err != nil || string(b) != "first" {
		t.Fatalf("first launch did not seed from shared: %q %v", b, err)
	}
	if !savesiso.HasJournal(mustModConfigJournal(t, inst.ID)) {
		t.Fatal("a live swap left no journal")
	}

	again := &deployment{}
	if err := svc.swapModConfig(t.Context(), vsGame, inst, pid, "", again); err != nil {
		t.Fatalf("a launch after a crashed one: %v", err)
	}
	again.unwind(t.Context())
	if b, err := safefs.ReadFile(filepath.Join(shared, "mod.json")); err != nil || string(b) != "first" {
		t.Fatalf("shared after recover and undo = %q %v", b, err)
	}

	testfs.WriteAt(t, filepath.Join(shared, "mod.json"), []byte("second"))
	seeded := &deployment{}
	if err := svc.swapModConfig(t.Context(), vsGame, inst, pid, "", seeded); err != nil {
		t.Fatal(err)
	}
	if b, err := safefs.ReadFile(filepath.Join(own, "mod.json")); err != nil || string(b) != "first" {
		t.Fatalf("a later launch re-seeded from shared: %q %v", b, err)
	}
	seeded.unwind(t.Context())
}

func TestSwapModConfigIgnoresAStatErrorOtherThanNotExist(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	svc, pid, _, inst := serverEnv(t)
	shared, err := game.PathFor(svc.home, svc.settings.Get(), vsGame, "", profile.ModConfigRole)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteAt(t, filepath.Join(shared, "mod.json"), []byte("shared"))
	dir, err := svc.profiles.ProfileDir(vsGame, pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("not a folder"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.swapModConfig(t.Context(), vsGame, inst, pid, "", &deployment{}); err == nil {
		t.Fatal("Apply must fail when the profile path is a file")
	}
	if b, err := safefs.ReadFile(filepath.Join(shared, "mod.json")); err != nil || string(b) != "shared" {
		t.Fatalf("a Stat error must not copy shared away: %q %v", b, err)
	}
}

func TestSwapModConfigUsesTheServerFolder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	svc, pid, _, inst := serverEnv(t)
	if _, err := svc.profiles.SetServerMode(vsGame, pid, true); err != nil {
		t.Fatal(err)
	}
	gameCfg, err := game.PathFor(svc.home, svc.settings.Get(), vsGame, "", profile.ModConfigRole)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteAt(t, filepath.Join(gameCfg, "mod.json"), []byte("client"))
	serverCfg, err := svc.serverModConfig(vsGame, pid)
	if err != nil {
		t.Fatal(err)
	}
	testfs.WriteAt(t, filepath.Join(serverCfg, "mod.json"), []byte("server"))
	dep := &deployment{}
	if err := svc.swapModConfig(t.Context(), vsGame, inst, pid, "", dep); err != nil {
		t.Fatal(err)
	}
	if b, err := safefs.ReadFile(filepath.Join(gameCfg, "mod.json")); err != nil || string(b) != "client" {
		t.Fatalf("server mode swapped the client folder: %q %v", b, err)
	}
	own := filepath.Join(mustProfileDir(t, svc, vsGame, pid), profile.ModConfigFolder, "mod.json")
	if b, err := safefs.ReadFile(own); err != nil || string(b) != "server" {
		t.Fatalf("own was not seeded from the server folder: %q %v", b, err)
	}
	dep.unwind(t.Context())
}

func TestRecoverDeploysFindsALeftoverJournalWhenTheProcessIsGone(t *testing.T) {
	svc, _ := startEnv(t)
	svc.procDir = t.TempDir()
	root := t.TempDir()
	mj := mustModConfigJournal(t, svc.selectedSlot(game.Find("stardew")).inst)
	if _, err := savesiso.Apply(mj, filepath.Join(root, "config"), filepath.Join(root, "prof"), true); err != nil {
		t.Fatal(err)
	}
	found, err := svc.RecoverDeploys(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(found, "stardew") {
		t.Fatalf("found = %v, want stardew", found)
	}
	if savesiso.HasJournal(mj) {
		t.Fatal("recovery left the journal")
	}
}

func TestSharedFilesListsNotesWhoseFilesAreStillThere(t *testing.T) {
	svc, _ := startEnv(t)
	svc.procDir = t.TempDir()
	sl := svc.selectedSlot(game.Find("stardew"))
	journal, err := journalDir(sl.inst)
	if err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(t.TempDir(), "too-large.bin")
	testfs.WriteAt(t, kept, []byte("abc"))
	gone := filepath.Join(t.TempDir(), "gone.bin")
	if err := datadir.WriteJSON(journal+".shared.json", []deploy.Note{
		{Path: kept, Size: 3, Kind: deploy.NoteTooLarge},
		{Path: gone, Size: 1, Kind: deploy.NoteRescued},
	}); err != nil {
		t.Fatal(err)
	}
	got := svc.SharedFiles("stardew")
	if len(got) != 1 || got[0].Path != kept || got[0].Size != 3 || got[0].Rescued {
		t.Fatalf("SharedFiles = %+v", got)
	}
}

func mustModConfigJournal(t *testing.T, installID string) string {
	t.Helper()
	j, err := modConfigJournal(installID)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func mustProfileDir(t *testing.T, svc *Service, gameID, profileID string) string {
	t.Helper()
	dir, err := svc.profiles.ProfileDir(gameID, profileID)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
