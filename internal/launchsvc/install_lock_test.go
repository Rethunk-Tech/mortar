//go:build !windows

package launchsvc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	svc.EnsureLoader = func(ctx context.Context, _, _ string, _ bool) error {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return errors.New("stop")
	}
	// Both launches fail once released; wait for them to finish, or their failure writes race the temp dir's removal.
	defer func() {
		close(release)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			svc.mu.Lock()
			n := len(svc.preparing)
			svc.mu.Unlock()
			if n == 0 && !svc.AnyBusy() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Error("the launches did not finish")
	}()
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

// fakeProc writes a /proc entry for pid under svc.procDir; an empty exe leaves /proc/<pid>/exe unreadable.
func fakeProc(t *testing.T, svc *Service, pid, exe string, args ...string) {
	t.Helper()
	dir := filepath.Join(svc.procDir, pid)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	cmdline := ""
	for _, a := range args {
		cmdline += a + "\x00"
	}
	if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o600); err != nil {
		t.Fatal(err)
	}
	if exe != "" {
		if err := os.Symlink(exe, filepath.Join(dir, "exe")); err != nil {
			t.Fatal(err)
		}
	}
}

// owners lists how many of the game's processes the folder install and the GOG install each see.
func owners(t *testing.T, pair installPair) (folder, gog int) {
	t.Helper()
	g := game.Find("stardew")
	f, err := pair.svc.gameProcs(slot{g, pair.folderID})
	if err != nil {
		t.Fatal(err)
	}
	o, err := pair.svc.gameProcs(slot{g, pair.gogID})
	if err != nil {
		t.Fatal(err)
	}
	return len(f), len(o)
}

func TestProcessOwnership(t *testing.T) {
	const wine = "/proton/files/bin/wine64-preloader"
	// {gog} is the GOG install's folder and {away} a game folder no install knows.
	cases := []struct {
		name, exe, arg string
		folder, gog    int
	}{
		{"copy outside every install is ignored", "{away}/StardewValley", "{away}/StardewValley", 0, 0},
		{"copy inside an install is that install's", "{gog}/StardewValley", "{gog}/StardewValley", 0, 1},
		{"unreadable exe is the selected install's", "", "{away}/StardewValley", 1, 0},
		{"Proton process is the install its Z: path names", wine, `Z:{gog}\Stardew Valley.exe`, 0, 1},
		{"Proton process outside every install is the selected install's", wine, `Z:{away}\Stardew Valley.exe`, 1, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pair := twoInstalls(t)
			pair.svc.procDir = t.TempDir()
			fill := strings.NewReplacer("{gog}", pair.gogDir, "{away}", filepath.Join(t.TempDir(), "Stardew Valley"))
			fakeProc(t, pair.svc, "7", fill.Replace(c.exe), fill.Replace(c.arg))
			if folder, gog := owners(t, pair); folder != c.folder || gog != c.gog {
				t.Fatalf("folder install sees %d, GOG install %d; want %d, %d", folder, gog, c.folder, c.gog)
			}
		})
	}
}
