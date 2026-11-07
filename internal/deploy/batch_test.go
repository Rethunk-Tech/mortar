package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

func TestATornUnloggedCopyIsRemovedAndNeverWrittenBackIntoTheProfile(t *testing.T) {
	rr := newRootRig(t)
	p := rr.plan()
	steps := sequence(t, func(r *rig) { r.apply() })
	_, fired := crashAt(slices.Index(steps, "flushed"), func() { _, _ = Place{}.Apply(t.Context(), p) })
	if !fired {
		t.Fatal("no crash at the flush")
	}
	torn := filepath.Join(rr.shared, "top.package")
	write(t, torn, "")
	if err := (Place{}).Recover(t.Context(), rr.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	if got := read(filepath.Join(rr.profile, "top.package")); got != "top" {
		t.Fatalf("the profile's copy = %q", got)
	}
	if read(torn) != "" || HasJournal(rr.view.JournalDir) {
		t.Fatal("the torn copy or the journal is still there")
	}
}

func BenchmarkApply2000Files(b *testing.B) {
	root := b.TempDir()
	if d := getenvDir(); d != "" {
		root = d
	}
	var files []launchplan.PlanFile
	for i := range 2000 {
		src := filepath.Join(root, "profile", fmt.Sprintf("d%d", i%20), fmt.Sprintf("f%d.package", i))
		writeB(b, src)
		files = append(files, launchplan.PlanFile{Src: src, Dst: filepath.Join("Mods", fmt.Sprintf("d%d", i%20), fmt.Sprintf("f%d.package", i))})
	}
	for n := 0; b.Loop(); n++ {
		install := filepath.Join(root, fmt.Sprint("game", n))
		view := View{JournalDir: filepath.Join(root, fmt.Sprint("journal", n))}
		p, err := Place{}.Plan(view, install, files)
		if err != nil {
			b.Fatal(err)
		}
		m, err := Place{}.Apply(b.Context(), p)
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		if err := (Place{}).Purge(b.Context(), m); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
	}
}

func getenvDir() string { return os.Getenv("MORTAR_BENCH_DIR") }

func writeB(b *testing.B, path string) {
	b.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("small mod file"), 0o600); err != nil {
		b.Fatal(err)
	}
}
