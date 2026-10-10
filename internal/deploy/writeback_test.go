package deploy

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

// rootRig places a profile's Mods folder into a shared folder outside the install, as a folder game does.
type rootRig struct {
	*rig
	profile, shared string
}

func newRootRig(t *testing.T) *rootRig {
	r := newRig(t)
	rr := &rootRig{rig: r, profile: filepath.Join(r.root, "profile", "Mods"), shared: filepath.Join(r.root, "docs", "Mods")}
	r.view.Roots = map[string]string{"mods": rr.shared}
	write(t, filepath.Join(rr.profile, "mc", "mc_settings.cfg"), "default=1")
	write(t, filepath.Join(rr.profile, "mc", "mc.ts4script"), "script")
	write(t, filepath.Join(rr.profile, "top.package"), "top")
	write(t, filepath.Join(rr.shared, "Resource.cfg"), "player's resource")
	r.files = nil
	for _, rel := range []string{"mc/mc_settings.cfg", "mc/mc.ts4script", "top.package"} {
		r.files = append(r.files, launchplan.PlanFile{Src: filepath.Join(rr.profile, filepath.FromSlash(rel)), Dst: filepath.FromSlash(rel), Root: "mods"})
	}
	return rr
}

func (rr *rootRig) purge(m Manifest) {
	rr.t.Helper()
	d, _ := Get(copyID)
	if err := d.Purge(rr.t.Context(), m); err != nil {
		rr.t.Fatal(err)
	}
}

func TestARewrittenProfileFileIsWrittenBackAndLeavesTheSharedFolder(t *testing.T) {
	rr := newRootRig(t)
	m := rr.apply()
	write(t, filepath.Join(rr.shared, "mc", "mc_settings.cfg"), "player=2")
	write(t, filepath.Join(rr.shared, "top.package"), "rewritten top")
	rr.purge(m)
	if got := read(filepath.Join(rr.profile, "mc", "mc_settings.cfg")); got != "player=2" {
		t.Fatalf("the profile's copy = %q", got)
	}
	if got := read(filepath.Join(rr.profile, "top.package")); got != "rewritten top" {
		t.Fatalf("a rewritten file at the root of the folder = %q", got)
	}
	if got := snapshot(t, rr.shared); !maps.Equal(got, map[string]string{"./": "", "Resource.cfg": "player's resource"}) {
		t.Fatalf("the shared folder after purge: %v", got)
	}
}

func TestARewrittenFileThatDisplacedThePlayersIsWrittenBackAndThePlayersReturns(t *testing.T) {
	rr := newRootRig(t)
	write(t, filepath.Join(rr.shared, "top.package"), "the player's own")
	m := rr.apply()
	write(t, filepath.Join(rr.shared, "top.package"), "rewritten top")
	rr.purge(m)
	if got := read(filepath.Join(rr.shared, "top.package")); got != "the player's own" {
		t.Fatalf("the shared file = %q", got)
	}
	if got := read(filepath.Join(rr.profile, "top.package")); got != "rewritten top" {
		t.Fatalf("the profile's copy = %q", got)
	}
}

func TestFilesCreatedDuringPlayAreAdoptedOnlyInFoldersAnEntryOwns(t *testing.T) {
	rr := newRootRig(t)
	old := filepath.Join(rr.shared, "mc", "players_own.cfg")
	write(t, old, "was here before")
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	m := rr.apply()
	write(t, filepath.Join(rr.shared, "mc", "mc_state.dat"), "created during play")
	write(t, filepath.Join(rr.shared, "mc", "deep", "x.dat"), "deep")
	write(t, filepath.Join(rr.shared, "root_created.dat"), "root file")
	write(t, filepath.Join(rr.shared, "unowned", "u.dat"), "unowned")
	rr.purge(m)
	for rel, want := range map[string]string{"mc/mc_state.dat": "created during play", "mc/deep/x.dat": "deep"} {
		if got := read(filepath.Join(rr.profile, filepath.FromSlash(rel))); got != want {
			t.Errorf("profile %s = %q", rel, got)
		}
	}
	got := snapshot(t, rr.shared)
	want := map[string]string{
		"./": "", "Resource.cfg": "player's resource", "root_created.dat": "root file",
		"unowned/": "", "unowned/u.dat": "unowned", "mc/": "", "mc/players_own.cfg": "was here before",
	}
	if !maps.Equal(got, want) {
		t.Fatalf("shared folder after purge:\n got %v\nwant %v", got, want)
	}
}

func TestNothingIsDeletedWhenTheProfileFolderIsGone(t *testing.T) {
	rr := newRootRig(t)
	m := rr.apply()
	write(t, filepath.Join(rr.shared, "mc", "mc_settings.cfg"), "player=2")
	write(t, filepath.Join(rr.shared, "mc", "mc_state.dat"), "created")
	if err := os.RemoveAll(filepath.Dir(rr.profile)); err != nil {
		t.Fatal(err)
	}
	rr.purge(m)
	if read(filepath.Join(rr.shared, "mc", "mc_settings.cfg")) != "player=2" || read(filepath.Join(rr.shared, "mc", "mc_state.dat")) != "created" {
		t.Fatal("a file with no profile to go to was deleted")
	}
}

func TestInstallRootFilesKeepTheirRule(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	ini := filepath.Join(r.install, "doorstop_config.ini")
	write(t, ini, "rewritten")
	write(t, filepath.Join(r.install, "created.dat"), "created")
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if read(ini) != "rewritten" || read(filepath.Join(r.install, "created.dat")) != "created" {
		t.Fatal("an install-root file was written back or adopted")
	}
	if read(filepath.Join(r.store, "doorstop_config.ini")) != "ini" {
		t.Fatal("the source was changed")
	}
}

// A crash at every step of purge, with a rewritten file, a rewritten file over a displaced one and a created file:
// recovery must end with the bytes in the profile and the shared folder as the player had it.
func TestACrashAtAnyWriteBackStepRecoversWithTheBytesInTheProfile(t *testing.T) {
	setup := func() (*rootRig, Manifest, map[string]string) {
		rr := newRootRig(t)
		write(t, filepath.Join(rr.shared, "top.package"), "the player's own")
		start := snapshot(t, rr.shared)
		m := rr.apply()
		write(t, filepath.Join(rr.shared, "mc", "mc_settings.cfg"), "player=2")
		write(t, filepath.Join(rr.shared, "top.package"), "rewritten top")
		write(t, filepath.Join(rr.shared, "mc", "mc_state.dat"), "created")
		return rr, m, start
	}
	rr, m, _ := setup()
	d, _ := Get(copyID)
	steps, _ := crashAt(-1, func() { _ = d.Purge(t.Context(), m) })
	if !slices.ContainsFunc(steps, func(s string) bool { return s == "adopted-copy:mc_state.dat" }) || !slices.Contains(steps, "written-back:2") {
		t.Fatalf("write-back steps missing: %v", steps)
	}
	_ = rr
	for i := range steps {
		rr, m, start := setup()
		got, fired := crashAt(i, func() { _ = d.Purge(t.Context(), m) })
		if !fired {
			t.Fatalf("the crash at purge step %d did not fire", i)
		}
		// Between a copy and its removal the bytes are in both places; recovery must still end clean.
		if err := d.Recover(t.Context(), rr.view.JournalDir, nil); err != nil {
			t.Fatalf("crash at %s: %v", got[i], err)
		}
		if read(filepath.Join(rr.profile, "mc", "mc_settings.cfg")) != "player=2" ||
			read(filepath.Join(rr.profile, "top.package")) != "rewritten top" ||
			read(filepath.Join(rr.profile, "mc", "mc_state.dat")) != "created" {
			t.Fatalf("crash at %s: the profile lost bytes", got[i])
		}
		if now := snapshot(t, rr.shared); !maps.Equal(now, start) {
			t.Fatalf("crash at %s: shared folder\n got %v\nwant %v", got[i], now, start)
		}
		if HasJournal(rr.view.JournalDir) {
			t.Fatalf("crash at %s: journal left", got[i])
		}
	}
}

// With the profile's folder gone and a changed file over a displaced one, a crash at any purge step still ends with the
// player's file back and the changed bytes in the rescue file.
func TestACrashAtAnyRescueStepKeepsTheChangedBytes(t *testing.T) {
	setup := func() (*rootRig, Manifest) {
		rr := newRootRig(t)
		write(t, filepath.Join(rr.shared, "top.package"), "the player's own")
		m := rr.apply()
		write(t, filepath.Join(rr.shared, "top.package"), "rewritten top")
		if err := os.RemoveAll(filepath.Dir(rr.profile)); err != nil {
			t.Fatal(err)
		}
		return rr, m
	}
	rr, m := setup()
	d, _ := Get(copyID)
	steps, _ := crashAt(-1, func() { _ = d.Purge(t.Context(), m) })
	if !slices.Contains(steps, "rescued:2") {
		t.Fatalf("no rescue step: %v", steps)
	}
	_ = rr
	for i := range steps {
		rr, m := setup()
		got, fired := crashAt(i, func() { _ = d.Purge(t.Context(), m) })
		if !fired {
			t.Fatalf("the crash at step %d did not fire", i)
		}
		if err := d.Recover(t.Context(), rr.view.JournalDir, nil); err != nil {
			t.Fatalf("crash at %s: %v", got[i], err)
		}
		top := filepath.Join(rr.shared, "top.package")
		if read(top) != "the player's own" || read(RescueName(top)) != "rewritten top" {
			t.Fatalf("crash at %s: player %q, rescue %q", got[i], read(top), read(RescueName(top)))
		}
	}
}

func TestFilesLeftSharedAreNotedUntilTheyAreGone(t *testing.T) {
	rr := newRootRig(t)
	write(t, filepath.Join(rr.shared, "top.package"), "the player's own")
	m := rr.apply()
	big := filepath.Join(rr.shared, "mc", "world.dat")
	write(t, big, "")
	if err := os.Truncate(big, maxAdoptFile+1); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(rr.shared, "top.package"), "rewritten top")
	if err := os.RemoveAll(filepath.Dir(rr.profile)); err != nil {
		t.Fatal(err)
	}
	// With the profile gone nothing is adopted either, so only the rescue is noted; bring the folder back for the cap.
	rr.purge(m)
	notes := Notes(rr.view.JournalDir)
	if len(notes) != 1 || notes[0].Kind != NoteRescued || notes[0].Path != RescueName(filepath.Join(rr.shared, "top.package")) {
		t.Fatalf("notes = %+v", notes)
	}
	if err := os.Remove(notes[0].Path); err != nil {
		t.Fatal(err)
	}
	if got := Notes(rr.view.JournalDir); len(got) != 0 {
		t.Fatalf("a note outlived its file: %+v", got)
	}

	rr2 := newRootRig(t)
	m2 := rr2.apply()
	big2 := filepath.Join(rr2.shared, "mc", "world.dat")
	write(t, big2, "")
	if err := os.Truncate(big2, maxAdoptFile+1); err != nil {
		t.Fatal(err)
	}
	rr2.purge(m2)
	n := Notes(rr2.view.JournalDir)
	if len(n) != 1 || n[0].Kind != NoteTooLarge || n[0].Size != maxAdoptFile+1 {
		t.Fatalf("notes = %+v", n)
	}
	if err := os.Remove(big2); err != nil {
		t.Fatal(err)
	}
	if len(Notes(rr2.view.JournalDir)) != 0 {
		t.Fatal("the note did not clear with its file")
	}
}
