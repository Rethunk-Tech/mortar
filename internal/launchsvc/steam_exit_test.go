//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

const protonGame = `Z:\home\nomad\.local\share\Steam\steamapps\common\Lethal Company\Lethal Company.exe`

// steamRun leaves the service where a Steam-relayed Lethal Company launch that Steam reported started leaves it: the
// game runs under Proton, Mortar is Running the profile and waits on a PID it did not start. With exits the wait
// ends when the game does; without, it never ends, as on a PID reused by another process.
func steamRun(t *testing.T, timedOutFirst, exits bool) (*Service, game.Game) {
	t.Helper()
	data := t.TempDir()
	profiles := profile.OpenIn(filepath.Join(data, "profiles"), store.OpenAt(filepath.Join(data, "store")))
	p := testenv.Profile(t, profiles, "lethal-company", "A")
	svc := NewService(t.TempDir(), nil, profiles)
	svc.procDir = t.TempDir()
	svc.WaitPID = func(pid int) (launch.Exit, error) {
		if !exits {
			select {}
		}
		for launchDirAlive(svc.procDir, pid) {
			time.Sleep(10 * time.Millisecond)
		}
		return launch.Exit{}, nil
	}
	g := game.Find("lethal-company")
	launching := func() {
		svc.mu.Lock()
		svc.logs[keyOf(g)] = session{buf: &launch.Buffer{}, profile: p.ID, started: time.Now()}
		svc.mu.Unlock()
		svc.set(Status{Game: g.ID(), State: Launching, Profile: p.ID, Since: time.Now().UnixMilli()})
		svc.watch(g)
	}
	if timedOutFirst {
		launching()
		svc.failStart(Status{Game: g.ID(), State: Failed, Profile: p.ID, Error: "the game did not start in time"})
	}
	launching()
	fakeProc(t, svc, "4242", "/proton/files/bin/wine64-preloader", protonGame)
	svc.set(Status{Game: g.ID(), State: Running, Profile: p.ID, Since: time.Now().UnixMilli()})
	svc.armReap(g)
	go svc.awaitPID(g)
	return svc, g
}

// idleWithin reports whether the game goes idle within limit while only the watcher polls.
func idleWithin(svc *Service, g game.Game, limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if svc.current(g).State == Idle {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func TestASteamRunClosesOnceTheGameIsGone(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                 string
		timedOutFirst, exits bool
	}{
		{"waited PID never exits", false, false},
		{"after a launch that timed out", true, false},
		{"waited PID exits", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			svc, g := steamRun(t, c.timedOutFirst, c.exits)
			time.Sleep(3 * pollEvery)
			if st := svc.current(g); st.State != Running {
				t.Fatalf("the game still runs, state = %+v", st)
			}
			if err := os.RemoveAll(filepath.Join(svc.procDir, "4242")); err != nil {
				t.Fatal(err)
			}
			if !idleWithin(svc, g, pollEvery+reapGrace+time.Second) {
				t.Fatalf("the run did not close after the game exited: %+v", svc.current(g))
			}
		})
	}
}

func TestStoppingARunWhoseGameIsGoneClosesIt(t *testing.T) {
	t.Parallel()
	svc, g := steamRun(t, true, false)
	if err := os.RemoveAll(filepath.Join(svc.procDir, "4242")); err != nil {
		t.Fatal(err)
	}
	if err := svc.Stop(t.Context(), g.ID()); err != nil {
		t.Fatal(err)
	}
	if st := svc.current(g); st.State != Idle {
		t.Fatalf("state = %+v", svc.current(g))
	}
}

func TestAGameStartedOutsideMortarAfterARunClosesToo(t *testing.T) {
	t.Parallel()
	svc, g := steamRun(t, false, true)
	proc := filepath.Join(svc.procDir, "4242")
	if err := os.RemoveAll(proc); err != nil {
		t.Fatal(err)
	}
	if !idleWithin(svc, g, time.Second) {
		t.Fatalf("state = %+v", svc.current(g))
	}
	fakeProc(t, svc, "4242", "/proton/files/bin/wine64-preloader", protonGame)
	if svc.poll(g); svc.current(g).State != Running {
		t.Fatalf("the game started from Steam is not seen: %+v", svc.current(g))
	}
	if err := os.RemoveAll(proc); err != nil {
		t.Fatal(err)
	}
	if svc.poll(g); svc.current(g).State != Idle {
		t.Fatalf("state = %+v", svc.current(g))
	}
}

// launchDirAlive reports whether the fake proc dir still lists pid.
func launchDirAlive(procDir string, pid int) bool {
	_, err := os.Stat(filepath.Join(procDir, strconv.Itoa(pid)))
	return err == nil
}

func TestABepInExGameIsTheProfileItsDoorstopTargetNames(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	_, profiles := testenv.Stores(t)
	a := testenv.Profile(t, profiles, "lethal-company", "A")
	b := testenv.Profile(t, profiles, "lethal-company", "B")
	dirB, err := profiles.ProfileDir("lethal-company", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, profiles)
	g := game.Find("lethal-company")
	for _, c := range []struct {
		name    string
		args    []string
		profile string
	}{
		{"Proton Z: target", bepinex5.LaunchArgs(dirB, 3, true), b.ID},
		{"native target", bepinex5.LaunchArgs(dirB, 4, false), b.ID},
		{"started from Steam without Mortar", nil, ""},
		{"vanilla", bepinex5.VanillaArgs(), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			svc.procDir = t.TempDir()
			fakeProc(t, svc, "4242", "/proton/files/bin/wine64-preloader", append([]string{protonGame}, c.args...)...)
			if svc.poll(g); svc.current(g).State != Running || svc.current(g).Profile != c.profile {
				t.Fatalf("status = %+v, want profile %q", svc.current(g), c.profile)
			}
			if svc.Running("lethal-company", a.ID) || svc.Running("lethal-company", b.ID) != (c.profile == b.ID) {
				t.Fatal("only the profile the target names is locked")
			}
			if err := os.RemoveAll(filepath.Join(svc.procDir, "4242")); err != nil {
				t.Fatal(err)
			}
			if svc.poll(g); svc.current(g).State != Idle {
				t.Fatalf("state = %+v", svc.current(g))
			}
		})
	}
}
