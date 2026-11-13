package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// play runs one Play with the mode and then its purge, which leaves the files in place for the next Play.
func (r *rig) play(t *testing.T, mode Mode) error {
	t.Helper()
	d := Place{Mode: mode}
	p, err := d.Plan(r.view, r.install, r.files)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	m, err := d.Apply(r.t.Context(), p)
	if err != nil {
		return err
	}
	return d.Purge(r.t.Context(), m)
}

func TestAReusedPlayKeepsTheFilesThatDidNotChange(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	mode := Mode{Reuse: ReuseStat}
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	proxy := filepath.Join(r.install, "winhttp.dll")
	before, err := os.Lstat(proxy)
	if err != nil {
		t.Fatal(err)
	}
	var reported []int
	mode.Progress = func(done, total int) {
		if done == total {
			reported = append(reported, total)
		}
	}
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	after, err := os.Lstat(proxy)
	if err != nil {
		t.Fatal(err)
	}
	// The kept file is the same file, not a fresh copy of it.
	if !os.SameFile(before, after) {
		t.Fatal("a file that did not change was placed again")
	}
	if len(reported) != 1 || reported[0] != 0 {
		t.Fatalf("the second Play placed %v files, want 0", reported)
	}
}

func TestAChangedPlacedFileIsPlacedAgain(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	mode := Mode{Reuse: ReuseStat}
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(r.install, "doorstop_config.ini")
	write(t, ini, "the settings a mod wrote, much longer than the original")
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	if got := read(ini); got != "ini" {
		t.Fatalf("doorstop_config.ini = %q, want the profile's copy back", got)
	}
	// The bytes written after the last Play's purge are no one's copy but the game folder's, so they are kept beside it.
	if got := read(RescueName(ini)); got != "the settings a mod wrote, much longer than the original" {
		t.Fatalf("rescue = %q, want the mod's settings", got)
	}
	if notes := Notes(r.view.JournalDir); len(notes) != 1 || notes[0].Path != RescueName(ini) || notes[0].Kind != NoteRescued {
		t.Fatalf("notes = %+v", notes)
	}
}

func TestATouchedButUnchangedPlacedFileIsPlacedAgainWithoutARescue(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	mode := Mode{Reuse: ReuseStat}
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(r.install, "doorstop_config.ini")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(ini, later, later); err != nil {
		t.Fatal(err)
	}
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	if got := read(ini); got != "ini" {
		t.Fatalf("doorstop_config.ini = %q", got)
	}
	if _, err := os.Lstat(RescueName(ini)); !os.IsNotExist(err) {
		t.Fatalf("a file with our bytes was rescued: %v", err)
	}
}

func TestAChangedSourceIsPlacedAgain(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	mode := Mode{Reuse: ReuseStat}
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(r.store, "doorstop_config.ini"), "ini, updated")
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	if got := read(filepath.Join(r.install, "doorstop_config.ini")); got != "ini, updated" {
		t.Fatalf("doorstop_config.ini = %q", got)
	}
}

func TestAKeepPurgeLeavesTheFilesInPlaceAndTakesBackTheChangedOne(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	write(t, filepath.Join(r.store, "extra.dll"), "extra")
	r.files = append(r.files, launchplan.PlanFile{Src: filepath.Join(r.store, "extra.dll"), Dst: "extra.dll"})

	if err := r.play(t, Mode{Reuse: ReuseStat}); err != nil {
		t.Fatal(err)
	}
	ini := filepath.Join(r.install, "doorstop_config.ini")
	write(t, ini, "the settings a mod wrote")
	found, _, ok, err := previous(r.view.JournalDir)
	if err != nil || !ok {
		t.Fatalf("the kept journal is gone: %v", err)
	}
	d := Place{Mode: Mode{Reuse: ReuseStat}}
	if err := d.Purge(t.Context(), found); err != nil {
		t.Fatal(err)
	}
	if got := read(ini); got != "the settings a mod wrote" {
		t.Fatalf("doorstop_config.ini = %q, want the mod's own bytes kept", got)
	}
	if got := read(filepath.Join(r.install, "winhttp.dll")); got != "proxy" {
		t.Fatalf("winhttp.dll = %q, want it still placed", got)
	}
	if _, err := os.Lstat(filepath.Join(r.install, "extra.dll")); err != nil {
		t.Fatalf("extra.dll was taken back: %v", err)
	}
	// A Play that places everything starts from the tree the first Play found, then puts its own files down.
	if err := r.play(t, Mode{Reuse: ReuseNone}); err != nil {
		t.Fatal(err)
	}
	if got := read(filepath.Join(r.install, "winhttp.dll")); got != "the player's own" {
		t.Fatalf("winhttp.dll = %q, want the player's own file back", got)
	}
}

func TestASmallerPlanTakesBackFilesTheLastPlayLeft(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	write(t, filepath.Join(r.store, "extra.dll"), "extra")
	r.files = append(r.files, launchplan.PlanFile{Src: filepath.Join(r.store, "extra.dll"), Dst: "extra.dll"})
	if err := r.play(t, Mode{Reuse: ReuseStat}); err != nil {
		t.Fatal(err)
	}
	r.files = r.files[:len(r.files)-1]
	if err := r.play(t, Mode{Reuse: ReuseStat}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(r.install, "extra.dll")); !os.IsNotExist(err) {
		t.Fatal("a file the new plan does not place was left in the install")
	}
	if got := read(filepath.Join(r.install, "winhttp.dll")); got != "proxy" {
		t.Fatalf("winhttp.dll = %q, want it still placed", got)
	}
}

func TestProgressReportsEachFilePlaced(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	var done, total int
	d := Place{Mode: Mode{Progress: func(d, t int) { done, total = d, t }}}
	p, err := d.Plan(r.view, r.install, r.files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if done != len(r.files) || total != len(r.files) {
		t.Fatalf("progress ended at %d of %d, want %d", done, total, len(r.files))
	}
}

func TestRecoverLeavesAKeptJournal(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	if err := r.play(t, Mode{Reuse: ReuseStat}); err != nil {
		t.Fatal(err)
	}
	if err := (Place{}).Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	if !HasJournal(r.view.JournalDir) {
		t.Fatal("startup recovery purged a kept journal")
	}
	if got := read(filepath.Join(r.install, "winhttp.dll")); got != "proxy" {
		t.Fatalf("winhttp.dll = %q, want it still placed", got)
	}
}

func TestAnOldJournalWithoutKeepFieldsStillRecoversAndPurges(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	before := testfs.Tree(t, r.install)
	r.apply()
	stripKeepFields(t, r.view.JournalDir)
	if err := (Place{}).Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	after := testfs.Tree(t, r.install)
	if len(after) != len(before) {
		t.Fatalf("tree differs\nbefore %v\n after %v", before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
	if HasJournal(r.view.JournalDir) {
		t.Fatal("the old journal was not taken back")
	}
}

func stripKeepFields(t *testing.T, dir string) {
	t.Helper()
	p := journalPath(dir)
	b, err := safefs.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw, "keep")
	delete(raw, "gen")
	ops, _ := raw["ops"].([]any)
	for _, op := range ops {
		m, _ := op.(map[string]any)
		for _, k := range []string{"srcSize", "srcMTime", "size", "mTime"} {
			delete(m, k)
		}
	}
	out, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

// A crash at any step of the Play that rescues a changed kept file, then a startup recovery and another Play, ends with
// the changed bytes in the rescue file, noted, and the profile's copy placed.
func TestACrashAtAnyPlayRescueStepKeepsTheChangedBytes(t *testing.T) {
	mode := Mode{Reuse: ReuseStat}
	setup := func() (*rig, string) {
		r := newRig(t)
		if err := r.play(t, mode); err != nil {
			t.Fatal(err)
		}
		ini := filepath.Join(r.install, "doorstop_config.ini")
		write(t, ini, "settings saved outside Mortar")
		return r, ini
	}
	r, _ := setup()
	steps, _ := crashAt(-1, func() { _ = r.play(t, mode) })
	// The steps before the rescue and after the copy are the crash tests' of any Play.
	from, to := slices.Index(steps, "play-rescue-logged:0"), slices.Index(steps, "copied:0")
	if from < 0 || to < from {
		t.Fatalf("no rescue step: %v", steps)
	}
	for i := from; i <= to; i++ {
		r, ini := setup()
		got, fired := crashAt(i, func() { _ = r.play(t, mode) })
		if !fired {
			t.Fatalf("the crash at step %d did not fire", i)
		}
		if err := (Place{}).Recover(t.Context(), r.view.JournalDir, nil); err != nil {
			t.Fatalf("crash at %s: %v", got[i], err)
		}
		if err := r.play(t, mode); err != nil {
			t.Fatalf("crash at %s: next Play: %v", got[i], err)
		}
		if read(ini) != "ini" || read(RescueName(ini)) != "settings saved outside Mortar" {
			t.Fatalf("crash at %s: placed %q, rescue %q", got[i], read(ini), read(RescueName(ini)))
		}
		if notes := Notes(r.view.JournalDir); len(notes) != 1 || notes[0].Path != RescueName(ini) {
			t.Fatalf("crash at %s: notes = %+v", got[i], notes)
		}
	}
}

// A kept file changed again after an earlier rescue keeps both edits: the second rescue never lands on the first.
func TestASecondRescueOfAKeptFileKeepsTheFirst(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	mode := Mode{Reuse: ReuseStat}
	ini := filepath.Join(r.install, "doorstop_config.ini")
	if err := r.play(t, mode); err != nil {
		t.Fatal(err)
	}
	for _, edit := range []string{"first edit outside Mortar", "second edit outside Mortar"} {
		write(t, ini, edit)
		if err := r.play(t, mode); err != nil {
			t.Fatal(err)
		}
	}
	var kept []string
	for _, n := range Notes(r.view.JournalDir) {
		kept = append(kept, read(n.Path))
	}
	slices.Sort(kept)
	if len(kept) != 2 || kept[0] != "first edit outside Mortar" || kept[1] != "second edit outside Mortar" {
		t.Fatalf("rescued = %q, want both edits", kept)
	}
}
