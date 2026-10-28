//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// A crashed game's process can outlive the started process's exit while Windows writes its report. The run stays open
// until it is gone, so the purge finds the files released and the poll does not take the dying process for a new run.
func TestARunStaysOpenWhileTheCrashedGameProcessLingers(t *testing.T) {
	t.Parallel()
	data := t.TempDir()
	_, profiles := testenv.StoresIn(t, data)
	a := testenv.Profile(t, profiles, "lethal-company", "A")
	svc := NewService(t.TempDir(), nil, profiles)
	svc.procDir = t.TempDir()
	g := game.Find("lethal-company")
	fakeProc(t, svc, "4242", "/proton/files/bin/wine64-preloader", protonGame)
	var purged atomic.Bool
	svc.mu.Lock()
	svc.logs[keyOf(g)] = session{buf: &launch.Buffer{}, profile: a.ID, started: time.Now(), deployed: &deployment{finish: func() { purged.Store(true) }}}
	svc.mu.Unlock()
	svc.set(Status{Game: g.ID(), State: Running, Profile: a.ID, Since: time.Now().UnixMilli()})

	go svc.finishWait(g, launch.Exit{Code: 3221225477})
	for range 5 {
		time.Sleep(pollEvery)
		if st := svc.current(g); st.State != Running || st.Profile != a.ID || purged.Load() {
			t.Fatalf("the run ended while the game process was alive: %+v, purged %v", st, purged.Load())
		}
	}
	if err := os.RemoveAll(filepath.Join(svc.procDir, "4242")); err != nil {
		t.Fatal(err)
	}
	if !idleWithin(svc, g, time.Second) || !purged.Load() {
		t.Fatalf("the run did not close once the process was gone: %+v, purged %v", svc.current(g), purged.Load())
	}
}
