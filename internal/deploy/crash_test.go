package deploy

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

type crashed struct{ step string }

// crashRig is the install of newRig plus a third file that needs a folder made for it; winhttp.dll displaces the
// player's own.
func crashRig(t *testing.T) *rig {
	r := newRig(t)
	write(t, filepath.Join(r.store, "extra.dll"), "extra")
	r.files = append(r.files, launchplan.PlanFile{Src: filepath.Join(r.store, "extra.dll"), Dst: "new/deeper/extra.dll"})
	return r
}

// crashAt runs fn, which panics with crashed when the step with this index in the run's sequence is reached; the
// steps seen so far are returned, and crashed is true when the crash fired.
func crashAt(index int, fn func()) (steps []string, crashed_ bool) {
	crashHook = func(name string) {
		steps = append(steps, name)
		if len(steps)-1 == index {
			panic(crashed{name})
		}
	}
	defer func() {
		crashHook = nil
		if v := recover(); v != nil {
			if _, ok := v.(crashed); !ok {
				panic(v)
			}
			crashed_ = true
		}
	}()
	fn()
	return steps, false
}

func sequence(t *testing.T, fn func(*rig)) []string {
	t.Helper()
	r := crashRig(t)
	steps, _ := crashAt(-1, func() { fn(r) })
	return steps
}

func (r *rig) leftover(t *testing.T, start map[string]string, step string) {
	t.Helper()
	d := Place{}
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatalf("after a crash at %s: recover: %v", step, err)
	}
	if got := testfs.Tree(t, r.install); !maps.Equal(got, start) {
		t.Fatalf("after a crash at %s the tree differs:\n got %v\nwant %v", step, got, start)
	}
	if HasJournal(r.view.JournalDir) {
		t.Fatalf("after a crash at %s the journal is still there", step)
	}
}

func TestACrashAfterAnyApplyStepIsRecoveredToTheStartingTree(t *testing.T) {
	steps := sequence(t, func(r *rig) { r.apply() })
	if len(steps) < 10 {
		t.Fatalf("only %d steps seen: %v", len(steps), steps)
	}
	for i := range steps {
		r := crashRig(t)
		start := testfs.Tree(t, r.install)
		p := r.plan()
		d := Place{}
		got, fired := crashAt(i, func() { _, _ = d.Apply(t.Context(), p) })
		if !fired {
			t.Fatalf("the crash at step %d (%s) did not fire", i, steps[i])
		}
		// The first partial copy a crash leaves behind.
		if got[i] == "before-copy:0" || got[i] == "before-copy:1" || got[i] == "before-copy:2" {
			var n int
			_, _ = fmt.Sscanf(got[i], "before-copy:%d", &n)
			write(t, datadir.CopyTempName(p.Ops[n].Dst), "partial")
		}
		r.leftover(t, start, got[i])
	}
}

func TestACrashAfterAnyPurgeStepIsRecoveredToTheStartingTree(t *testing.T) {
	steps := sequence(t, func(r *rig) {
		m := r.apply()
		d := Place{}
		_ = d.Purge(t.Context(), m)
	})
	purge := steps[slices.Index(steps, "logged:2")+1:]
	if len(purge) < 5 {
		t.Fatalf("only %d purge steps seen: %v", len(purge), steps)
	}
	applySteps := len(steps) - len(purge)
	for i := range purge {
		r := crashRig(t)
		start := testfs.Tree(t, r.install)
		m := r.apply()
		d := Place{}
		got, fired := crashAt(i, func() { _ = d.Purge(t.Context(), m) })
		if !fired {
			t.Fatalf("the crash at purge step %d (%s) did not fire (apply took %d steps)", i, purge[i], applySteps)
		}
		r.leftover(t, start, got[i])
	}
}

// A placed file the game rewrote is Mortar's only where it displaced the player's file; with nothing displaced it holds
// data that is no longer ours and stays.
func TestRecoveryOfARewrittenPlacedFileFollowsWhetherSomethingWasDisplaced(t *testing.T) {
	t.Parallel()
	r := crashRig(t)
	start := testfs.Tree(t, r.install)
	r.apply()
	write(t, filepath.Join(r.install, "winhttp.dll"), "rewritten by the game")
	ini := filepath.Join(r.install, "doorstop_config.ini")
	write(t, ini, "settings the mod wrote")
	d := Place{}
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	if got := read(filepath.Join(r.install, "winhttp.dll")); got != "the player's own" {
		t.Fatalf("a rewritten file that displaced the player's: %q", got)
	}
	if got := read(ini); got != "settings the mod wrote" {
		t.Fatalf("a rewritten file with nothing displaced was removed or changed: %q", got)
	}
	delete(start, "doorstop_config.ini")
	want := testfs.Tree(t, r.install)
	delete(want, "doorstop_config.ini")
	if !maps.Equal(want, start) {
		t.Fatalf("tree differs apart from the kept file:\n got %v\nwant %v", want, start)
	}
}

func TestRecoveryNeverDeletesAFileOfThePlayersThatWasNotPlaced(t *testing.T) {
	r := crashRig(t)
	p := r.plan()
	d := Place{}
	// Crash before the first copy: nothing is placed, and the player puts a file where one would go.
	_, _ = crashAt(slices.Index(sequence(t, func(r *rig) { r.apply() }), "before-copy:1"), func() { _, _ = d.Apply(t.Context(), p) })
	write(t, p.Ops[1].Dst, "the player's new file")
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	if got := read(p.Ops[1].Dst); got != "the player's new file" {
		t.Fatalf("the player's file = %q", got)
	}
}

// A crash in a keep purge that gave the player's file back, then a startup recovery and another Play, still ends, once
// the journal is taken back, with the player's file where it was and no rescue of it.
func TestACrashInAKeepPurgeThatGaveThePlayersFileBackIsRecovered(t *testing.T) {
	mode := Mode{Reuse: ReuseStat}
	dll := func(r *rig) string { return filepath.Join(r.install, "winhttp.dll") }
	play := func(r *rig) {
		d := Place{Mode: mode}
		p, err := d.Plan(r.view, r.install, r.files)
		if err != nil {
			t.Fatal(err)
		}
		m, err := d.Apply(t.Context(), p)
		if err != nil {
			t.Fatal(err)
		}
		write(t, dll(r), "rewritten by the game")
		_ = d.Purge(t.Context(), m)
	}
	steps := sequence(t, play)
	from := slices.Index(steps, "logged:2") + 1
	for i := from; i < len(steps); i++ {
		r := crashRig(t)
		start := testfs.Tree(t, r.install)
		got, fired := crashAt(i, func() { play(r) })
		if !fired {
			t.Fatalf("the crash at step %d did not fire", i)
		}
		if err := (Place{}).Recover(t.Context(), r.view.JournalDir, nil); err != nil {
			t.Fatalf("crash at %s: recover: %v", got[i], err)
		}
		if err := r.play(t, mode); err != nil {
			t.Fatalf("crash at %s: next Play: %v", got[i], err)
		}
		if err := (Place{}).DiscardKept(t.Context(), r.view.JournalDir); err != nil {
			t.Fatalf("crash at %s: discard: %v", got[i], err)
		}
		if tree := testfs.Tree(t, r.install); !maps.Equal(tree, start) {
			t.Fatalf("crash at %s: tree differs:\n got %v\nwant %v", got[i], tree, start)
		}
	}
}
