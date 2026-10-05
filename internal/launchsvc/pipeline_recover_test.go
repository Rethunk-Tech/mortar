//go:build !windows

package launchsvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/deploy"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/savesiso"
)

// crashedLaunch leaves what a launch that died mid-game leaves: a deploy journal over a displaced player's file, and
// a save swap, both under the journals of the slot's install.
type crashedLaunch struct {
	install, dll, saves, prof, deployDir, savesDir string
	placed                                         deploy.Manifest
	swapped                                        savesiso.Manifest
}

func leaveCrashedLaunch(t *testing.T, svc *Service) crashedLaunch {
	t.Helper()
	sl := svc.selectedSlot(game.Find("stardew"))
	root := t.TempDir()
	c := crashedLaunch{install: filepath.Join(root, "game"), saves: filepath.Join(root, "Saves"), prof: filepath.Join(root, "prof")}
	c.dll = filepath.Join(c.install, "winhttp.dll")
	src := filepath.Join(root, "store", "winhttp.dll")
	for path, body := range map[string]string{c.dll: "the player's own", src: "proxy", filepath.Join(c.saves, "shared.sav"): "shared"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var err error
	if c.deployDir, err = journalDir(sl.inst); err != nil {
		t.Fatal(err)
	}
	if c.savesDir, err = savesJournal(sl.inst); err != nil {
		t.Fatal(err)
	}
	d, _ := deploy.Get(deployerID)
	plan, err := d.Plan(deploy.View{JournalDir: c.deployDir}, c.install, []launchplan.PlanFile{{Src: src, Dst: "winhttp.dll"}})
	if err != nil {
		t.Fatal(err)
	}
	if c.placed, err = d.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if c.swapped, err = savesiso.Apply(c.savesDir, c.saves, c.prof, true); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRecoverGameDeploysPutsBackWhatACrashedLaunchChanged(t *testing.T) {
	svc, _ := startEnv(t)
	svc.procDir = t.TempDir()
	c := leaveCrashedLaunch(t, svc)
	found, err := svc.RecoverGameDeploys(context.Background(), "stardew")
	if err != nil || !found {
		t.Fatalf("found %v, err %v", found, err)
	}
	if b, _ := fsx.ReadFile(c.dll); string(b) != "the player's own" {
		t.Fatalf("the player's file = %q", b)
	}
	if b, _ := fsx.ReadFile(filepath.Join(c.saves, "shared.sav")); string(b) != "shared" {
		t.Fatalf("the shared saves = %q", b)
	}
	if deploy.HasJournal(c.deployDir) || savesiso.HasJournal(c.savesDir) {
		t.Fatal("a journal is left after recovery")
	}
	if found, _ := svc.RecoverGameDeploys(context.Background(), "stardew"); found {
		t.Fatal("recovery found a journal that was already finished")
	}
}

func TestRecoverGameDeploysLeavesAnActiveLaunchAlone(t *testing.T) {
	svc, p := startEnv(t)
	svc.procDir = t.TempDir()
	c := leaveCrashedLaunch(t, svc)
	svc.set(Status{Game: "stardew", Install: svc.selectedSlot(game.Find("stardew")).inst, State: Launching, Profile: p.ID})
	if _, err := svc.RecoverGameDeploys(context.Background(), "stardew"); err != nil {
		t.Fatal(err)
	}
	if !deploy.HasJournal(c.deployDir) || !savesiso.HasJournal(c.savesDir) {
		t.Fatal("the journals of a running launch were recovered")
	}
	if left := svc.LeftoverJournals("stardew"); len(left) != 0 {
		t.Fatalf("a running launch's journals are leftovers: %v", left)
	}
}

func TestLeftoverJournalsNamesBothJournalsOfAnIdleInstall(t *testing.T) {
	svc, _ := startEnv(t)
	svc.procDir = t.TempDir()
	if left := svc.LeftoverJournals("stardew"); len(left) != 0 {
		t.Fatalf("leftovers with nothing crashed: %v", left)
	}
	if left := svc.LeftoverJournals("nosuchgame"); left != nil {
		t.Fatalf("leftovers of an unknown game: %v", left)
	}
	c := leaveCrashedLaunch(t, svc)
	left := svc.LeftoverJournals("stardew")
	if len(left) != 2 || left[0] != c.deployDir || left[1] != c.savesDir {
		t.Fatalf("leftovers = %v, want the deploy and saves journals", left)
	}
}

func TestUnwindTakesBackTheDeployAndTheSaveSwap(t *testing.T) {
	svc, _ := startEnv(t)
	c := leaveCrashedLaunch(t, svc)
	d, _ := deploy.Get(deployerID)
	dep := &deployment{d: d, m: c.placed, saves: &c.swapped}
	dep.unwind(context.Background())
	dep.unwind(context.Background())
	var nilDep *deployment
	nilDep.unwind(context.Background())
	if b, _ := fsx.ReadFile(c.dll); string(b) != "the player's own" {
		t.Fatalf("the player's file = %q", b)
	}
	if b, _ := fsx.ReadFile(filepath.Join(c.saves, "shared.sav")); string(b) != "shared" {
		t.Fatalf("the shared saves = %q", b)
	}
	if deploy.HasJournal(c.deployDir) || savesiso.HasJournal(c.savesDir) {
		t.Fatal("a journal is left after unwind")
	}
}

func TestRecoverGameDeploysFindsEitherJournal(t *testing.T) {
	if found, err := mustService(t).RecoverGameDeploys(context.Background(), "nosuchgame"); found || err != nil {
		t.Fatalf("an unknown game: found %v, err %v", found, err)
	}
	for _, only := range []string{"deploy", "saves"} {
		svc := mustService(t)
		c := leaveCrashedLaunch(t, svc)
		if only == "deploy" {
			if err := savesiso.Purge(c.swapped); err != nil {
				t.Fatal(err)
			}
		} else {
			d, _ := deploy.Get(deployerID)
			if err := d.Purge(context.Background(), c.placed); err != nil {
				t.Fatal(err)
			}
		}
		found, err := svc.RecoverGameDeploys(context.Background(), "stardew")
		if err != nil || !found {
			t.Fatalf("only the %s journal: found %v, err %v", only, found, err)
		}
		if deploy.HasJournal(c.deployDir) || savesiso.HasJournal(c.savesDir) {
			t.Fatalf("only the %s journal: one is left", only)
		}
	}
}

func mustService(t *testing.T) *Service {
	t.Helper()
	svc, _ := startEnv(t)
	svc.procDir = t.TempDir()
	return svc
}

func TestAnUnreadableProcessListCountsAsInUse(t *testing.T) {
	svc := mustService(t)
	c := leaveCrashedLaunch(t, svc)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc.procDir = blocker
	if _, err := svc.RecoverGameDeploys(context.Background(), "stardew"); err != nil {
		t.Fatal(err)
	}
	if !deploy.HasJournal(c.deployDir) || !savesiso.HasJournal(c.savesDir) {
		t.Fatal("journals were recovered while the game's processes could not be read")
	}
}

func TestAFailedDeployTakesBackWhatItPlaced(t *testing.T) {
	_ = mustService(t)
	root := t.TempDir()
	inst := game.Install{ID: "install-a", Dir: filepath.Join(root, "game")}
	good, missing := filepath.Join(root, "good.dll"), filepath.Join(root, "missing.dll")
	for path, body := range map[string]string{good: "proxy", filepath.Join(inst.Dir, "a.dll"): "the player's own"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan := &launchplan.Plan{Files: []launchplan.PlanFile{{Src: good, Dst: "a.dll"}}}
	// Canceled once the journal is written: the deploy fails with its record on disk and nothing placed yet.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := startDeploy(ctx, inst, plan); err == nil {
		t.Fatal("a canceled deploy succeeded")
	}
	if dir, _ := journalDir(inst.ID); deploy.HasJournal(dir) {
		t.Fatal("the canceled deploy left its journal")
	}
	missingPlan := &launchplan.Plan{Files: []launchplan.PlanFile{{Src: missing, Dst: "b.dll"}}}
	if _, err := startDeploy(context.Background(), inst, missingPlan); err == nil {
		t.Fatal("a deploy with a missing source succeeded")
	}
	if b, _ := fsx.ReadFile(filepath.Join(inst.Dir, "a.dll")); string(b) != "the player's own" {
		t.Fatalf("the player's file after a failed deploy = %q", b)
	}
	if dir, _ := journalDir(inst.ID); deploy.HasJournal(dir) {
		t.Fatal("the failed deploy left its journal")
	}
}

func TestSwapSavesGivesTheProfileItsOwnAndUndoesIt(t *testing.T) {
	svc, p := startEnv(t)
	svc.procDir = t.TempDir()
	g := game.Find("stardew")
	if _, err := svc.profiles.SetSeparateSaves("stardew", p.ID, true, false); err != nil {
		t.Fatal(err)
	}
	shared, err := game.SavesDir(svc.home, svc.settings.Get(), "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(shared, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(shared, "shared.sav"), []byte("shared"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, selected := svc.installs(g)
	dep := &deployment{}
	if err := svc.swapSaves(context.Background(), "stardew", selected, p.ID, "", dep); err != nil {
		t.Fatal(err)
	}
	if dep.saves == nil || dep.finish == nil {
		t.Fatalf("the swap is not tracked for undoing: %+v", dep)
	}
	if _, err := os.Stat(filepath.Join(shared, "shared.sav")); err == nil {
		t.Fatal("the game still sees the shared saves")
	}
	// A crash leaves the swap; the next launch finishes it before making its own.
	again := &deployment{}
	if err := svc.swapSaves(context.Background(), "stardew", selected, p.ID, "", again); err != nil {
		t.Fatalf("a launch after a crashed one: %v", err)
	}
	again.unwind(context.Background())
	if b, _ := fsx.ReadFile(filepath.Join(shared, "shared.sav")); string(b) != "shared" {
		t.Fatalf("the shared saves after the undo = %q", b)
	}
	// A profile that keeps no saves of its own is not swapped.
	if _, err := svc.profiles.SetSeparateSaves("stardew", p.ID, false, false); err != nil {
		t.Fatal(err)
	}
	none := &deployment{}
	if err := svc.swapSaves(context.Background(), "stardew", selected, p.ID, "", none); err != nil || none.saves != nil {
		t.Fatalf("a shared-saves profile was swapped: %v %+v", err, none)
	}
}
