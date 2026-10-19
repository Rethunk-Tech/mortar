package queue

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

func bigPackage(t *testing.T, size, free int64) *fixture {
	t.Helper()
	f := newFixture(t)
	c, _ := f.s.d.Client()
	f.s.d.Closure = func(_ context.Context, _ string, _ []thunderstore.Ref, _ bool) ([]thunderstore.Resolved, error) {
		return []thunderstore.Resolved{{Namespace: "Me", Name: "Big", Version: "1.0.0", URL: c.BaseURL + "/cdn/file.zip", Size: size}}, nil
	}
	f.s.d.InstallPackage = f.s.d.Install
	f.s.d.FreeBytes = func(string) (int64, error) { return free, nil }
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{{Kind: KindInstall, Game: "riskofrain2", Profile: "p1", Package: "Me-Big"}}); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestALargeDownloadWaitsForContinue(t *testing.T) {
	f := bigPackage(t, LargeDownloadBytes+1<<20, 1<<50)
	st := f.wait("the confirmation", f.item(StateConfirmLarge))
	if st.Items[0].SizeKB != (LargeDownloadBytes+1<<20)>>10 || len(f.installs) != 0 {
		t.Fatalf("item %+v installs %v", st.Items[0], f.installs)
	}
	f.s.ConfirmLarge(st.Items[0].ID)
	f.wait("done", f.item(StateDone))
	if len(f.installs) != 1 {
		t.Fatalf("installs %v", f.installs)
	}
}

func TestCancellingALargeDownloadEndsIt(t *testing.T) {
	f := bigPackage(t, LargeDownloadBytes+1<<20, 1<<50)
	st := f.wait("the confirmation", f.item(StateConfirmLarge))
	f.s.Cancel(st.Items[0].ID)
	f.wait("cancelled", f.item(StateCancelled))
	if len(f.installs) != 0 {
		t.Fatalf("installs %v", f.installs)
	}
}

func TestADownloadAtTheThresholdDoesNotAsk(t *testing.T) {
	f := bigPackage(t, LargeDownloadBytes, 1<<50)
	f.wait("done", f.item(StateDone))
}

func TestADownloadThatCannotFitFailsBeforeAnyByteWithNoContinue(t *testing.T) {
	size := LargeDownloadBytes + 1<<20
	f := bigPackage(t, size, 3*size)
	st := f.wait("failed", f.item(StateFailed))
	it := st.Items[0]
	if !strings.Contains(it.Error, "Not enough disk space") || !strings.Contains(it.Error, "GB") || it.ErrorKind != FailDisk || len(f.installs) != 0 {
		t.Fatalf("item %+v", it)
	}
	f.leftovers()
}

func TestCheckSpaceSplitsVolumesAndSkipsAnUnknownSize(t *testing.T) {
	dir, dl := t.TempDir(), t.TempDir()
	free := map[string]int64{}
	s := &Service{d: Deps{Dir: dir, DownloadDir: func() string { return dl }, FreeBytes: func(p string) (int64, error) { return free[p], nil }}}
	size := int64(1 << 30)
	item := Item{ID: "x", FileName: "a.zip"}
	free[dl], free[dir] = size+spaceMargin, 2*size+spaceMargin
	if err := s.checkSpace(item, size); err != nil {
		t.Fatalf("both volumes have room: %v", err)
	}
	free[dir]--
	var ns *NoSpaceError
	if err := s.checkSpace(item, size); !errors.As(err, &ns) || ns.Path != dir || ns.Need != 2*size+spaceMargin {
		t.Fatalf("store volume: %v", err)
	}
	free[dir], free[dl] = 1<<50, size+spaceMargin-1
	if err := s.checkSpace(item, size); !errors.As(err, &ns) || ns.Path != dl {
		t.Fatalf("downloads volume: %v", err)
	}
	if err := s.checkSpace(item, 0); err != nil {
		t.Fatalf("an unknown size is not checked: %v", err)
	}
}
