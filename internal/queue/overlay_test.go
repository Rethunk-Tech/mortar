package queue

import (
	"archive/zip"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func TestOptionalFileWaitsForItsMainFile(t *testing.T) {
	main := &Item{ID: "a", Game: "stardew", Profile: "p", ModID: 7, State: StateDownloading}
	opt := &Item{ID: "b", Game: "stardew", Profile: "p", ModID: 7, State: StateQueued}
	other := &Item{ID: "c", Game: "stardew", Profile: "p", ModID: 8, State: StateQueued}
	s := &Service{items: []*Item{main, opt, other}}
	if !s.waitsForSameMod(opt) || s.waitsForSameMod(main) || s.waitsForSameMod(other) {
		t.Fatal("only the later file of the same mod waits")
	}
	main.State = StateFailed
	if s.waitsForSameMod(opt) {
		t.Fatal("a finished main file still holds its optional file")
	}
	if got := failureKind(&profile.InstallError{Msg: "x", Err: &profile.NoBaseError{Archive: "x.zip"}}); got != FailNoBase {
		t.Fatalf("kind = %q", got)
	}
}

func TestManifestLess(t *testing.T) {
	write := func(name string, files ...string) string {
		p := filepath.Join(t.TempDir(), name)
		f, err := fsx.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		zw := zip.NewWriter(f)
		for _, n := range files {
			w, _ := zw.Create(n)
			_, _ = w.Write([]byte(`{"Name":"A","UniqueID":"a.b","Version":"1.0.0"}`))
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
		return p
	}
	if !manifestLess(write("opt.zip", "Mod/assets/a.png")) || manifestLess(write("main.zip", "Mod/manifest.json")) {
		t.Fatal("manifestLess")
	}
}
