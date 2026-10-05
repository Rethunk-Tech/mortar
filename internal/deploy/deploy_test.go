package deploy

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

type rig struct {
	t                    *testing.T
	root, install, store string
	view                 View
	inst                 InstallView
	pkgA, pkgB           Package
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

func newRig(t *testing.T) *rig {
	root := t.TempDir()
	r := &rig{t: t, root: root, install: filepath.Join(root, "game"), store: filepath.Join(root, "store")}
	r.view = View{JournalDir: filepath.Join(root, "journal"), Overwrite: filepath.Join(root, "overwrite")}
	r.inst = InstallView{Dir: r.install, Targets: []Target{
		{ID: "mods", Root: filepath.Join(r.install, "Mods")},
		{ID: "saves", Root: filepath.Join(r.install, "Saves"), Writable: true, Home: filepath.Join(root, "home-saves")},
	}}
	write(t, filepath.Join(r.install, "Mods", "keep.txt"), "user's own")
	write(t, filepath.Join(r.install, "Mods", "a.txt"), "original a")
	write(t, filepath.Join(r.store, "A", "a.txt"), "from A")
	write(t, filepath.Join(r.store, "A", "only.txt"), "only A")
	write(t, filepath.Join(r.store, "A", "slot.sav"), "slot")
	write(t, filepath.Join(r.store, "B", "a.txt"), "from B")
	r.pkgA = Package{ID: "A", Root: filepath.Join(r.store, "A"), Layout: installer.Layout{Files: []installer.File{
		{Src: "a.txt", Target: "mods", Rel: "a.txt"}, {Src: "only.txt", Target: "mods", Rel: "sub/only.txt"}, {Src: "slot.sav", Target: "saves", Rel: "slot.sav"},
	}}}
	r.pkgB = Package{ID: "B", Root: filepath.Join(r.store, "B"), Layout: installer.Layout{Files: []installer.File{{Src: "a.txt", Target: "mods", Rel: "a.txt"}}}}
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
			b, _ := fsx.ReadFile(p)
			out[rel] = string(b)
		}
		return nil
	})
	return out
}

func (r *rig) apply() Manifest {
	r.t.Helper()
	d, _ := Get("link-into-install")
	plan, err := d.Plan(r.view, r.inst, []Package{r.pkgA, r.pkgB}, []launchplan.PlanFile{{Src: filepath.Join(r.store, "A", "only.txt"), Dst: "loader.dll"}})
	if err != nil {
		r.t.Fatal(err)
	}
	m, err := d.Apply(r.t.Context(), plan)
	if err != nil {
		r.t.Fatal(err)
	}
	return m
}

func TestPlanOrderAndApplyPurgeRoundTrip(t *testing.T) {
	r := newRig(t)
	before := snapshot(t, r.install)
	d, _ := Get("link-into-install")
	plan, _ := d.Plan(r.view, r.inst, []Package{r.pkgA, r.pkgB}, nil)
	if len(plan.Conflicts) != 1 || plan.Conflicts[0].Winners[0] != "B" || plan.Conflicts[0].Losers[0] != "A" {
		t.Fatalf("conflicts %+v", plan.Conflicts)
	}
	m := r.apply()
	if got := snapshot(t, r.install)["Mods/a.txt"]; got != "from B" {
		t.Fatalf("later package must win, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(r.view.JournalDir, journalFile)); err != nil {
		t.Fatalf("manifest not persisted: %v", err)
	}
	// Links for plain targets, copies for writable ones, so the game cannot write through to the store.
	saved, _ := os.Stat(filepath.Join(r.install, "Saves", "slot.sav"))
	sav, _ := os.Stat(filepath.Join(r.store, "A", "slot.sav"))
	if os.SameFile(saved, sav) {
		t.Fatal("a writable target was linked")
	}
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	after := snapshot(t, r.install)
	delete(after, "Saves/") // the install had no Saves folder; Purge removes the one Apply made
	if len(after) != len(before) {
		t.Fatalf("tree differs\nbefore %v\n after %v", before, after)
	}
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
}

func TestHarvestRoutesWhatTheGameWrote(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	write(t, filepath.Join(r.install, "Saves", "slot.sav"), "slot, played")
	write(t, filepath.Join(r.install, "Saves", "new.sav"), "new save")
	write(t, filepath.Join(r.install, "Mods", "log.txt"), "game log")
	write(t, filepath.Join(r.install, "Mods", "keep.txt"), "user's own, edited") // not ours: untouched
	d, _ := Get("link-into-install")
	changes, err := d.Harvest(t.Context(), m)
	if err != nil || len(changes) != 3 {
		t.Fatalf("changes %+v %v", changes, err)
	}
	for path, want := range map[string]string{
		filepath.Join(r.root, "home-saves", "slot.sav"):       "slot, played",
		filepath.Join(r.root, "home-saves", "new.sav"):        "new save",
		filepath.Join(r.root, "overwrite", "mods", "log.txt"): "game log",
	} {
		if b, _ := fsx.ReadFile(path); string(b) != want {
			t.Errorf("%s = %q, want %q", path, b, want)
		}
	}
	if b, _ := fsx.ReadFile(filepath.Join(r.install, "Mods", "keep.txt")); string(b) != "user's own, edited" {
		t.Fatal("harvest touched a file that was never Mortar's")
	}
	// What was moved is gone; only the placed file the game changed, which is copied, shows up again.
	if again, _ := d.Harvest(t.Context(), m); len(again) != 1 || again[0].Kind != "changed" {
		t.Fatalf("second harvest found %+v", again)
	}
}

func TestRecoverAfterACrash(t *testing.T) {
	r := newRig(t)
	before := snapshot(t, r.install)
	r.apply()
	write(t, filepath.Join(r.install, "Saves", "late.sav"), "written before the crash")
	d, _ := Get("link-into-install")
	// The game is still running: leave everything.
	if err := d.Recover(t.Context(), r.view.JournalDir, func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.install, "Mods", "a.txt")); err != nil || snapshot(t, r.install)["Mods/a.txt"] != "from B" {
		t.Fatal("recover touched a running game's files")
	}
	if err := d.Recover(t.Context(), r.view.JournalDir, func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	if b, _ := fsx.ReadFile(filepath.Join(r.root, "home-saves", "late.sav")); string(b) != "written before the crash" {
		t.Fatal("recover lost the game's save")
	}
	after := snapshot(t, r.install)
	delete(after, "Saves/")
	for k, v := range before {
		if after[k] != v {
			t.Fatalf("%s: %q, want %q", k, after[k], v)
		}
	}
	if _, err := os.Stat(r.view.JournalDir); !os.IsNotExist(err) {
		t.Fatal("journal kept after recovery")
	}
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatalf("recover with no journal: %v", err)
	}
}
