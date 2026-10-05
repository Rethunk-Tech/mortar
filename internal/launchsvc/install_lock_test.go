//go:build !windows

package launchsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// twoInstalls adds a GOG install beside startEnv's folder install and returns both install ids.
type installPair struct {
	svc                     *Service
	profA, profB            string
	folderID, gogID, gogDir string
}

func twoInstalls(t *testing.T) installPair {
	t.Helper()
	svc, a0 := startEnv(t)
	gogRoot := t.TempDir()
	gogDir := filepath.Join(gogRoot, "Stardew Valley")
	if err := os.MkdirAll(gogDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gogDir, "Stardew Valley.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.settings.Update(func(v *settings.Settings) {
		v.LauncherRoots = map[string][]string{game.LauncherGOG: {gogRoot}}
	}); err != nil {
		t.Fatal(err)
	}
	g := game.Find("stardew")
	all, selected := svc.installs(g)
	if len(all) != 1 || all[0].Store != game.StoreGOG {
		t.Fatalf("installs = %+v", all)
	}
	b0, err := svc.profiles.Create("stardew", "B")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.profiles.SetInstall("stardew", b0.ID, all[0].ID); err != nil {
		t.Fatal(err)
	}
	return installPair{svc, a0.ID, b0.ID, selected.ID, all[0].ID, gogDir}
}

func TestTwoInstallsOfOneGameAreClaimedSideBySide(t *testing.T) {
	pair := twoInstalls(t)
	svc, a, b, folderID, gogID := pair.svc, pair.profA, pair.profB, pair.folderID, pair.gogID
	if folderID == gogID {
		t.Fatal("the installs share an id")
	}
	release := make(chan struct{})
	svc.EnsureLoader = func(ctx context.Context, _ string, _ bool) error {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return errors.New("stop")
	}
	defer close(release)
	if err := svc.Start(context.Background(), "stardew", a, false); err != nil {
		t.Fatal(err)
	}
	if err := svc.Start(context.Background(), "stardew", b, false); err != nil {
		t.Fatalf("a second install must be claimable while the first prepares: %v", err)
	}
	if err := svc.Start(context.Background(), "stardew", a, false); err == nil {
		t.Fatal("a second launch on the same install must be refused")
	}
	svc.mu.Lock()
	n := len(svc.preparing)
	svc.mu.Unlock()
	if n != 2 {
		t.Fatalf("claims = %d, want one per install", n)
	}
}

func TestBusyAndStopFollowTheInstall(t *testing.T) {
	pair := twoInstalls(t)
	svc, folderID, gogID := pair.svc, pair.folderID, pair.gogID
	svc.set(Status{Game: "stardew", Install: gogID, State: Launching, Profile: "p"})
	if !svc.BusyInstall(gogID) || svc.BusyInstall(folderID) || !svc.Busy("stardew") || !svc.AnyBusy() {
		t.Fatal("busy must follow the install that runs, and stay true for the game")
	}
	if err := svc.StopInstall(folderID); err == nil {
		t.Fatal("stopped an install that is not running")
	}
	if err := svc.StopInstall(gogID); err == nil {
		t.Fatal("stopped an install that has not started running")
	}
	svc.set(Status{Game: "stardew", Install: gogID, State: Idle})
	if svc.BusyInstall(gogID) || svc.AnyBusy() {
		t.Fatal("an idle game kept its install busy")
	}
}

func TestProtonProcessIsAttributedThroughItsZDrivePath(t *testing.T) {
	pair := twoInstalls(t)
	svc, folderID, gogID, gogDir := pair.svc, pair.folderID, pair.gogID, pair.gogDir
	svc.procDir = t.TempDir()
	pid := filepath.Join(svc.procDir, "7")
	if err := os.MkdirAll(pid, 0o700); err != nil {
		t.Fatal(err)
	}
	winPath := `Z:` + filepath.ToSlash(gogDir) + `/Stardew Valley.exe`
	for i := range winPath {
		if winPath[i] == '/' {
			winPath = winPath[:i] + `\` + winPath[i+1:]
		}
	}
	if err := os.WriteFile(filepath.Join(pid, "cmdline"), []byte(winPath+"\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	g := game.Find("stardew")
	if ps, err := svc.gameProcs(slot{g, gogID}); err != nil || len(ps) != 1 {
		t.Fatalf("the install the path names owns the process: %v, %v", ps, err)
	}
	if ps, err := svc.gameProcs(slot{g, folderID}); err != nil || len(ps) != 0 {
		t.Fatalf("the other install must not see it: %v, %v", ps, err)
	}
	if st := svc.statusOf(slot{g, gogID}); st.State != Running || st.Install != gogID {
		t.Fatalf("status = %+v", st)
	}
	if svc.BusyInstall(folderID) {
		t.Fatal("the other install is not busy")
	}
}
