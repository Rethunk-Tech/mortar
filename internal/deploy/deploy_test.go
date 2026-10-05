package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

type rig struct {
	t                    *testing.T
	root, install, store string
	view                 View
	files                []launchplan.PlanFile
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(path string) string {
	b, _ := fsx.ReadFile(path)
	return string(b)
}

// newRig is an install with a game file, a file of the player's that a placed file displaces, and two loader files.
func newRig(t *testing.T) *rig {
	root := t.TempDir()
	r := &rig{t: t, root: root, install: filepath.Join(root, "game"), store: filepath.Join(root, "store")}
	r.view = View{JournalDir: filepath.Join(root, "journal")}
	write(t, filepath.Join(r.install, "game.exe"), "exe")
	write(t, filepath.Join(r.install, "winhttp.dll"), "the player's own")
	write(t, filepath.Join(r.store, "winhttp.dll"), "proxy")
	write(t, filepath.Join(r.store, "doorstop_config.ini"), "ini")
	r.files = []launchplan.PlanFile{
		{Src: filepath.Join(r.store, "winhttp.dll"), Dst: "winhttp.dll"},
		{Src: filepath.Join(r.store, "doorstop_config.ini"), Dst: "doorstop_config.ini"},
	}
	return r
}

// snapshot is every file and folder of the install with its content.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, _ error) error {
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			out[rel+"/"] = ""
		} else {
			out[rel] = read(p)
		}
		return nil
	})
	return out
}

func (r *rig) plan() Plan {
	r.t.Helper()
	d, _ := Get(copyID)
	p, err := d.Plan(r.view, r.install, r.files)
	if err != nil {
		r.t.Fatal(err)
	}
	return p
}

func (r *rig) apply() Manifest {
	r.t.Helper()
	d, _ := Get(copyID)
	m, err := d.Apply(r.t.Context(), r.plan())
	if err != nil {
		r.t.Fatal(err)
	}
	return m
}

func TestApplyPurgeRestoresTheInstall(t *testing.T) {
	r := newRig(t)
	before := snapshot(t, r.install)
	m := r.apply()
	if read(filepath.Join(r.install, "winhttp.dll")) != "proxy" || read(filepath.Join(r.install, "doorstop_config.ini")) != "ini" {
		t.Fatal("the loader files are not in the install")
	}
	if _, err := os.Stat(filepath.Join(r.view.JournalDir, journalFile)); err != nil {
		t.Fatalf("manifest not persisted: %v", err)
	}
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	after := snapshot(t, r.install)
	if len(after) != len(before) {
		t.Fatalf("tree differs\nbefore %v\n after %v", before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
	if _, err := os.Stat(r.view.JournalDir); !os.IsNotExist(err) {
		t.Fatal("journal kept after purge")
	}
}

func TestPlanRefusesAFileOutsideTheInstall(t *testing.T) {
	r := newRig(t)
	d, _ := Get(copyID)
	if _, err := d.Plan(r.view, r.install, []launchplan.PlanFile{{Src: "x", Dst: "../escape.dll"}}); err == nil {
		t.Fatal("a destination outside the install was planned")
	}
}

func TestPurgeLeavesTheGamesOwnFilesAlone(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	write(t, filepath.Join(r.install, "game.exe"), "patched by Steam")
	write(t, filepath.Join(r.install, "Data", "new.assets"), "added by an update")
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if read(filepath.Join(r.install, "game.exe")) != "patched by Steam" || read(filepath.Join(r.install, "Data", "new.assets")) != "added by an update" {
		t.Fatal("purge touched a file that was never Mortar's")
	}
}

func TestRecoverAfterACrash(t *testing.T) {
	r := newRig(t)
	before := snapshot(t, r.install)
	r.apply()
	d, _ := Get(copyID)
	// The game is still running: leave everything.
	if err := d.Recover(t.Context(), r.view.JournalDir, func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	if read(filepath.Join(r.install, "winhttp.dll")) != "proxy" {
		t.Fatal("recover touched a running game's files")
	}
	if err := d.Recover(t.Context(), r.view.JournalDir, func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	after := snapshot(t, r.install)
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
	if len(after) != len(before) {
		t.Fatalf("tree differs\nbefore %v\n after %v", before, after)
	}
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatalf("recover with no journal: %v", err)
	}
}

func TestApplyRefusesAnUnrecoveredJournal(t *testing.T) {
	r := newRig(t)
	r.apply()
	d, _ := Get(copyID)
	if _, err := d.Apply(t.Context(), r.plan()); !errors.Is(err, ErrUnrecovered) {
		t.Fatalf("second apply: %v", err)
	}
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	if read(filepath.Join(r.install, "winhttp.dll")) != "the player's own" {
		t.Fatal("the player's file was not restored")
	}
	m := r.apply()
	if err := d.Purge(t.Context(), m); err != nil || read(filepath.Join(r.install, "winhttp.dll")) != "the player's own" {
		t.Fatalf("purge after the retry: %v", err)
	}
}

func TestPurgeKeepsTheFileOfAnOpThatNeverRan(t *testing.T) {
	r := newRig(t)
	dst := filepath.Join(r.install, "winhttp.dll")
	// The player's file happens to equal ours, and the cancelled launch never got to displace or replace it.
	write(t, dst, "proxy")
	hash, _ := fsx.SHA256(dst)
	m := Manifest{Dir: r.install, View: r.view, Ops: []Placed{{Dst: dst, Src: filepath.Join(r.store, "winhttp.dll"), Hash: hash, Displaced: filepath.Join(r.view.JournalDir, "displaced", "0")}}}
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil || read(dst) != "proxy" {
		t.Fatalf("purge removed a file it never placed: %q, %v", read(dst), err)
	}
}

func TestACanceledApplyRecordsWhatItPlaced(t *testing.T) {
	r := newRig(t)
	d, _ := Get(copyID)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	m, err := d.Apply(ctx, r.plan())
	if err == nil || len(m.Ops) != 2 || m.Ops[0].Done {
		t.Fatalf("apply: %v, ops %+v", err, m.Ops)
	}
	if err := d.Purge(t.Context(), m); err != nil || read(filepath.Join(r.install, "winhttp.dll")) != "the player's own" {
		t.Fatalf("purge after a canceled apply: %v", err)
	}
}

func TestAPlacedFileIsACopyOfTheSource(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	// The game rewrites a placed file in place; the profile's own copy must not change with it.
	f, err := os.OpenFile(filepath.Join(r.install, "winhttp.dll"), os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("rewritten by the game")
	_ = f.Close()
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if got := read(filepath.Join(r.store, "winhttp.dll")); got != "proxy" {
		t.Fatalf("the source is now %q", got)
	}
}

func TestPurgeRunAgainKeepsWhatItRestored(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read-only folder does not stop a delete on Windows")
	}
	r := newRig(t)
	r.files[1].Dst = filepath.Join("sub", "doorstop_config.ini")
	m := r.apply()
	sub := filepath.Join(r.install, "sub")
	if err := fsx.Chmod(sub, 0o500); err != nil {
		t.Fatal(err)
	}
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err == nil {
		t.Fatal("the purge removed a file from a read-only folder")
	}
	if err := fsx.Chmod(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	if got := read(filepath.Join(r.install, "winhttp.dll")); got != "the player's own" {
		t.Fatalf("player's winhttp.dll after the second purge = %q", got)
	}
	if _, err := os.Lstat(filepath.Join(sub, "doorstop_config.ini")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the placed file is still there: %v", err)
	}
}

func TestPurgeLeavesAFileThatIsNoLongerOurs(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	ini := filepath.Join(r.install, "doorstop_config.ini")
	write(t, ini, "the player's edit")
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if got := read(ini); got != "the player's edit" {
		t.Fatalf("doorstop_config.ini = %q", got)
	}
}

func TestRecoverKeepsAJournalItCannotRead(t *testing.T) {
	r := newRig(t)
	r.apply()
	write(t, filepath.Join(r.view.JournalDir, journalFile), "{not json")
	d, _ := Get(copyID)
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err == nil {
		t.Fatal("a damaged journal was taken as recovered")
	}
	if left, _ := os.ReadDir(filepath.Join(r.view.JournalDir, "displaced")); len(left) != 1 {
		t.Fatalf("the player's set-aside file is gone: %v", left)
	}
}

func TestACanceledPurgeLeavesTheJournalToFinishLater(t *testing.T) {
	r := newRig(t)
	before := snapshot(t, r.install)
	m := r.apply()
	d, _ := Get(copyID)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := d.Purge(ctx, m); err == nil {
		t.Fatal("a canceled purge reported success")
	}
	// A journal that names no folder of its own is finished from the folder it was read from.
	m.View.JournalDir = ""
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(r.view.JournalDir, journalFile), string(b))
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	after := snapshot(t, r.install)
	if len(after) != len(before) || after["winhttp.dll"] != "the player's own" {
		t.Fatalf("tree differs\nbefore %v\n after %v", before, after)
	}
	if _, err := os.Stat(r.view.JournalDir); !os.IsNotExist(err) {
		t.Fatal("journal kept after recover")
	}
}

func TestApplyMarksEveryPlacedFileDone(t *testing.T) {
	r := newRig(t)
	for _, op := range r.apply().Ops {
		if !op.Done {
			t.Fatalf("%s was placed but is not marked done", op.Dst)
		}
	}
}

func TestPurgeRemovesTheFoldersApplyMade(t *testing.T) {
	r := newRig(t)
	write(t, filepath.Join(r.store, "deep.ini"), "deep")
	// The parent folder is made for the first file and the child for the second, so Created lists the parent first.
	r.files = append(r.files,
		launchplan.PlanFile{Src: filepath.Join(r.store, "deep.ini"), Dst: "sub/a.ini"},
		launchplan.PlanFile{Src: filepath.Join(r.store, "deep.ini"), Dst: "sub/deep/deep.ini"})
	before := snapshot(t, r.install)
	m := r.apply()
	if want := []string{filepath.Join(r.install, "sub"), filepath.Join(r.install, "sub", "deep")}; !slices.Equal(m.Created, want) {
		t.Fatalf("created = %v, want only the folders apply made: %v", m.Created, want)
	}
	if read(filepath.Join(r.install, "sub", "deep", "deep.ini")) != "deep" {
		t.Fatal("the file is not in its new folder")
	}
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, r.install); len(after) != len(before) {
		t.Fatalf("folders left behind\nbefore %v\n after %v", before, after)
	}
}

func TestApplyFailsWhenAFolderCannotBeMade(t *testing.T) {
	r := newRig(t)
	write(t, filepath.Join(r.install, "blocker"), "a file where a folder is needed")
	write(t, filepath.Join(r.store, "x.ini"), "x")
	r.files = []launchplan.PlanFile{{Src: filepath.Join(r.store, "x.ini"), Dst: "blocker/x.ini"}}
	d, _ := Get(copyID)
	if _, err := d.Apply(t.Context(), r.plan()); err == nil {
		t.Fatal("apply reported success without placing the file")
	}
}

func TestHasJournalFollowsTheRecord(t *testing.T) {
	r := newRig(t)
	if HasJournal(r.view.JournalDir) {
		t.Fatal("a journal before any apply")
	}
	r.apply()
	if !HasJournal(r.view.JournalDir) {
		t.Fatal("no journal after apply")
	}
}

func TestAFailedPurgeRecordsWhatItAlreadyUndid(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	// Ops undo last to first. The first op's destination is now a folder with content, which cannot be removed.
	blocked := m.Ops[0].Dst
	if err := os.Remove(blocked); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(blocked, "keep"), "x")
	write(t, filepath.Join(r.view.JournalDir, "displaced", "kept"), "the player's own")
	m.Ops[0].Displaced = filepath.Join(r.view.JournalDir, "displaced", "kept")
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err == nil {
		t.Fatal("purge reported success past a destination it could not clear")
	}
	var saved Manifest
	if err := json.Unmarshal([]byte(read(filepath.Join(r.view.JournalDir, journalFile))), &saved); err != nil {
		t.Fatal(err)
	}
	undone := 0
	for _, op := range saved.Ops {
		if op.Undone {
			undone++
		}
	}
	if undone != 1 {
		t.Fatalf("the journal must name the one op already undone: %+v", saved.Ops)
	}
}
