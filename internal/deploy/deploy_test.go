package deploy

import (
	"io/fs"
	"os"
	"path/filepath"
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
