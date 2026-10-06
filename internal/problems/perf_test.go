//go:build unix

package problems

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/modfixture"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// The budgets are CPU time, about twice what each path took on the maintainer's 811-mod profile when they were set;
// CPU time holds still on a busy machine where wall time does not. The race detector multiplies every figure by five
// or more, so the budgets run only in the plain build that MORTAR_PERF=1 asks for.
const (
	// warmCheckBudget is a whole restarted process: start, open the stores, one Problems check over warm disk caches.
	warmCheckBudget = 1200 * time.Millisecond
	// packEditBudget is the Content Patcher check after the biggest pack's content.json changed.
	packEditBudget = 800 * time.Millisecond
	// driftBudget is one walk of the profile's mods folder against its recorded snapshot.
	driftBudget = 500 * time.Millisecond
)

const (
	// perfEnv turns the budgets on.
	perfEnv = "MORTAR_PERF"
	// perfRealEnv names a real profile (game/id) in the data folder XDG_DATA_HOME points at, a sandbox copy, to measure
	// in place of the fixture; calibrating the fixture compares the two.
	perfRealEnv = "MORTAR_PERF_REAL"
	// perfChildEnv makes TestPerfWarmCheckChild the restarted Mortar of TestPerfBudgets, checking game/id.
	perfChildEnv = "MORTAR_PERF_CHILD"
)

func cpuNow() time.Duration {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

func measure(f func()) (wall, cpu time.Duration) {
	c, w := cpuNow(), time.Now()
	f()
	return time.Since(w), cpuNow() - c
}

// bestOf runs f n times and keeps the run with the least CPU time, so one run slowed by the machine's other work does
// not fail a budget.
func bestOf(n int, f func() (wall, cpu time.Duration)) (wall, cpu time.Duration) {
	for i := range n {
		w, c := f()
		if i == 0 || c < cpu {
			wall, cpu = w, c
		}
	}
	return wall, cpu
}

func within(t *testing.T, label string, wall, cpu, budget time.Duration) {
	t.Helper()
	t.Logf("%s: wall %v, cpu %v (budget %v)", label, wall.Round(time.Millisecond), cpu.Round(time.Millisecond), budget)
	if cpu > budget {
		t.Errorf("%s took %v of CPU, over its budget of %v", label, cpu.Round(time.Millisecond), budget)
	}
}

func perfService(t testing.TB) *Service {
	t.Helper()
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	s := NewService(t.TempDir(), set, profiles, nil)
	// No network: the budgets cover what Mortar computes, not how fast Nexus or smapi.io answer.
	s.meta = fakeMeta{}
	return s
}

// forgetCaches drops every Content Patcher cache, in memory and on disk, so the next check reads every pack.
func forgetCaches(t testing.TB) {
	t.Helper()
	framework.Forget()
	dir, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"problems-content-patcher", "problems-content-patcher-parts.json.gz", "problems-content-packs.json", "problems-map-scans.json"} {
		if err := os.RemoveAll(filepath.Join(dir, "cache", name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPerfBudgets(t *testing.T) {
	gameID, id, onReal := strings.Cut(os.Getenv(perfRealEnv), "/")
	if !onReal && os.Getenv(perfEnv) != "1" {
		t.Skip("set MORTAR_PERF=1 (a plain build, not -race) to hold the checks to their budgets")
	}
	if !onReal {
		t.Setenv("XDG_DATA_HOME", testfs.DataHome(t))
	}
	s := perfService(t)
	if !onReal {
		gameID = "stardew"
		wall, cpu := measure(func() { id = modfixture.Profile(t, s.profiles, modfixture.Real) })
		t.Logf("fixture written: wall %v, cpu %v", wall.Round(time.Millisecond), cpu.Round(time.Millisecond))
	}

	forgetCaches(t)
	var r Result
	wall, cpu := measure(func() {
		var err error
		if r, err = s.Problems(t.Context(), gameID, id); err != nil {
			t.Fatal(err)
		}
	})
	t.Logf("cold check: wall %v, cpu %v, %d asset conflicts", wall.Round(time.Millisecond), cpu.Round(time.Millisecond), len(r.AssetConflicts))
	for _, x := range r.Timings {
		t.Logf("  %s: %dms over %d", x.Name, x.Ms, x.Count)
	}

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	wall, cpu = bestOf(3, func() (time.Duration, time.Duration) {
		child := exec.CommandContext(t.Context(), self, "-test.run=^TestPerfWarmCheckChild$", "-test.count=1")
		child.Env = append(os.Environ(), perfChildEnv+"="+gameID+"/"+id)
		start := time.Now()
		if out, err := child.CombinedOutput(); err != nil {
			t.Fatalf("restarted check: %v\n%s", err, out)
		}
		return time.Since(start), child.ProcessState.UserTime() + child.ProcessState.SystemTime()
	})
	within(t, "warm check after a restart", wall, cpu, warmCheckBudget)

	mods, err := s.installed(gameID, id)
	if err != nil {
		t.Fatal(err)
	}
	in := framework.Input{Enabled: slices.DeleteFunc(slices.Clone(mods), func(m framework.Mod) bool { return !m.Enabled }), All: mods}
	contentpatcher.Driver{}.Analyze(in)
	pack := biggestPack(t, mods)
	wall, cpu = bestOf(3, func() (time.Duration, time.Duration) {
		touchPack(t, pack)
		return measure(func() { contentpatcher.Driver{}.Analyze(in) })
	})
	within(t, "Content Patcher after one pack edit", wall, cpu, packEditBudget)

	if _, err := s.profiles.ScanModsDrift(gameID, id); err != nil {
		t.Fatal(err)
	}
	wall, cpu = bestOf(3, func() (time.Duration, time.Duration) {
		return measure(func() {
			if _, err := s.profiles.ScanModsDrift(gameID, id); err != nil {
				t.Fatal(err)
			}
		})
	})
	within(t, "drift walk", wall, cpu, driftBudget)
}

// TestPerfWarmCheckChild is the check a restarted Mortar runs: nothing in memory, the disk caches warm.
func TestPerfWarmCheckChild(t *testing.T) {
	gameID, id, ok := strings.Cut(os.Getenv(perfChildEnv), "/")
	if !ok {
		t.Skip("run by TestPerfBudgets")
	}
	if _, err := perfService(t).Problems(t.Context(), gameID, id); err != nil {
		t.Fatal(err)
	}
}

// biggestPack is the content.json of the enabled pack with the most text there, the edit that re-reads the most. The
// file is put back after the test.
func biggestPack(t *testing.T, mods []framework.Mod) string {
	t.Helper()
	path, size := "", int64(-1)
	for _, m := range mods {
		if !m.Enabled || !strings.EqualFold(m.ContentPackFor, "Pathoschild.ContentPatcher") {
			continue
		}
		if info, err := os.Stat(filepath.Join(m.Folder, "content.json")); err == nil && info.Size() > size {
			path, size = filepath.Join(m.Folder, "content.json"), info.Size()
		}
	}
	if path == "" {
		t.Fatal("no enabled Content Patcher pack")
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = fsx.WriteFile(path, raw, info.Mode())
		_ = os.Chtimes(path, info.ModTime(), info.ModTime())
	})
	return path
}

// touchPack appends a blank line to content.json, dated past the window in which a fresh stamp is not trusted.
func touchPack(t *testing.T, path string) {
	t.Helper()
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-time.Minute)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}
