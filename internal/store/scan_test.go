package store

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/avscan"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

type fakeScanner struct {
	hit   *avscan.Detection
	err   error
	calls int
}

func (f *fakeScanner) Scan(context.Context, string) (avscan.Detection, bool, error) {
	f.calls++
	if f.hit != nil {
		return *f.hit, true, f.err
	}
	return avscan.Detection{}, false, f.err
}

func scanned(t *testing.T, f *fakeScanner) (*Store, *[]error) {
	t.Helper()
	s := newStore(t)
	var failures []error
	s.SetScanner(func() avscan.Scanner { return f }, func(_, _ string, err error) { failures = append(failures, err) })
	return s, &failures
}

func TestADetectionRefusesTheItemAndLeavesNothingInTheStore(t *testing.T) {
	f := &fakeScanner{hit: &avscan.Detection{Name: "Test.Threat", File: "Mod/a.dll", Scanner: "fake"}}
	s, _ := scanned(t, f)
	p := buildZip(t, map[string]string{"Mod/a.dll": "x"})
	_, err := s.AddArchive(t.Context(), "stardew", p)
	var det *DetectedError
	if !errors.As(err, &det) || det.Name != "Test.Threat" || det.File != "Mod/a.dll" || det.Key == "" {
		t.Fatalf("err = %v", err)
	}
	if usererr.KindOf(err) != usererr.Malware || !strings.Contains(err.Error(), "Test.Threat in Mod/a.dll") {
		t.Fatalf("kind = %v, text = %q", usererr.KindOf(err), err)
	}
	if got := names(t, s.root+"/"+blobsDir); len(got) != 0 {
		t.Fatalf("extracted files were kept: %v", got)
	}
}

func TestInstallAnywayLetsTheItemInOnce(t *testing.T) {
	f := &fakeScanner{hit: &avscan.Detection{Name: "Test.Threat"}}
	s, _ := scanned(t, f)
	p := buildZip(t, map[string]string{"Mod/a.dll": "x"})
	key, err := hashKey(p)
	if err != nil {
		t.Fatal(err)
	}
	s.AllowUnscanned("stardew", key, Override{})
	if _, err := s.AddArchive(t.Context(), "stardew", p); err != nil {
		t.Fatalf("allowed install failed: %v", err)
	}
	if f.calls != 0 {
		t.Fatalf("the scanner ran %d times for an allowed item", f.calls)
	}
	if err := s.AddArchiveKey(t.Context(), "stardew", "other-key", buildZip(t, map[string]string{"b": "y"})); err == nil {
		t.Fatal("the allowance covered another item")
	}
}

func TestAScannerErrorInstallsAndIsReported(t *testing.T) {
	f := &fakeScanner{err: errors.New("clamd went away")}
	s, failures := scanned(t, f)
	if _, err := s.AddArchive(t.Context(), "stardew", buildZip(t, map[string]string{"a": "x"})); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if len(*failures) != 1 || !strings.Contains((*failures)[0].Error(), "clamd") {
		t.Fatalf("failures = %v", *failures)
	}
}

func TestNoScannerInstallsSilently(t *testing.T) {
	f := &fakeScanner{err: avscan.ErrNoScanner}
	s, failures := scanned(t, f)
	if _, err := s.AddArchive(t.Context(), "stardew", buildZip(t, map[string]string{"a": "x"})); err != nil || len(*failures) != 0 {
		t.Fatalf("err = %v, failures = %v", err, *failures)
	}
}

func TestLoaderBundlesAreNotScanned(t *testing.T) {
	f := &fakeScanner{hit: &avscan.Detection{Name: "Heuristic"}}
	s, _ := scanned(t, f)
	dir := t.TempDir()
	if err := s.AddDir(t.Context(), "lethal-company", LoaderKey("bepinex5", "5.4.2"), dir); err != nil {
		t.Fatalf("loader bundle was scanned: %v", err)
	}
	if f.calls != 0 {
		t.Fatalf("scanner ran %d times", f.calls)
	}
}

type ctxScanner struct{}

func (ctxScanner) Scan(ctx context.Context, _ string) (avscan.Detection, bool, error) {
	<-ctx.Done()
	return avscan.Detection{}, false, ctx.Err()
}

func TestCancellingTheInstallStopsTheScanAndAddsNothing(t *testing.T) {
	s := newStore(t)
	s.SetScanner(func() avscan.Scanner { return ctxScanner{} }, nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	p := buildZip(t, map[string]string{"a": "x"})
	key, err := s.AddArchive(ctx, "stardew", p)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if _, perr := s.Path("stardew", key); perr == nil {
		t.Fatal("a cancelled install left the item in the store")
	}
}

func TestRealTimeBlockReportsAVirusErrnoAndAVanishedStagingFileAsMalware(t *testing.T) {
	staging := t.TempDir()
	gone := &fs.PathError{Op: "open", Path: filepath.Join(staging, "a.zip"), Err: fs.ErrNotExist}
	elsewhere := &fs.PathError{Op: "open", Path: filepath.Join(t.TempDir(), "a.zip"), Err: fs.ErrNotExist}
	for name, c := range map[string]struct {
		err  error
		want bool
	}{
		"vanished in staging":     {gone, true},
		"missing outside staging": {elsewhere, false},
		"other error":             {errors.New("disk on fire"), false},
	} {
		got := RealTimeBlock("stardew", "k1", c.err, staging)
		var det *DetectedError
		isMalware := usererr.KindOf(got) == usererr.Malware && errors.As(got, &det) && det.Removed && det.Scanner == RealTimeScanner && det.Name == ""
		if isMalware != c.want {
			t.Errorf("%s: malware=%v, want %v (%v)", name, isMalware, c.want, got)
		}
	}
	for _, errno := range []syscall.Errno{225, 226} {
		if !isVirusErrno(&fs.PathError{Op: "write", Path: "x", Err: errno}) {
			t.Errorf("errno %d is not recognised as a virus error", errno)
		}
	}
	if isVirusErrno(syscall.ENOENT) {
		t.Error("ENOENT is not a virus error")
	}
	already := usererr.Wrap(usererr.Malware, &DetectedError{Key: "k"})
	if !errors.Is(RealTimeBlock("stardew", "k1", already, staging), already) {
		t.Error("a detection already reported was rewrapped")
	}
}

func TestAnOverrideIsHandedOverOnlyWhenItsInstallSucceeds(t *testing.T) {
	f := &fakeScanner{hit: &avscan.Detection{Name: "Test.Threat", Scanner: "fake"}}
	s, _ := scanned(t, f)
	ov := Override{Profile: "p1", Name: "mod.zip", Detection: "Test.Threat"}
	fill := func(tmp string) error { return os.WriteFile(filepath.Join(tmp, "a"), []byte("x"), 0o600) }
	root := filepath.Join(s.root, "stardew")

	// The scan is skipped, then the rename into place fails because the final folder is not empty.
	final := filepath.Join(root, "blocked")
	if err := os.MkdirAll(final, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(final, "x"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.AllowUnscanned("stardew", "blocked", ov)
	if err := s.install(t.Context(), "stardew", "blocked", final, fill, func() int64 { return 0 }); err == nil {
		t.Fatal("the install into a non-empty folder succeeded")
	}
	if _, ok := s.TakeOverride("stardew", "blocked"); ok {
		t.Fatal("an override survived a failed install")
	}

	s.AllowUnscanned("stardew", "fine", ov)
	if err := s.install(t.Context(), "stardew", "fine", filepath.Join(root, "fine"), fill, func() int64 { return 0 }); err != nil {
		t.Fatal(err)
	}
	if got, ok := s.TakeOverride("stardew", "fine"); !ok || got != ov {
		t.Fatalf("override = %+v, %v", got, ok)
	}
	if _, ok := s.TakeOverride("stardew", "fine"); ok {
		t.Fatal("an override was handed over twice")
	}
}

func TestARealTimeRemovalDuringInstallKeepsItsMalwareKindAndKey(t *testing.T) {
	s := newStore(t)
	final := filepath.Join(s.root, "stardew", "gone")
	fill := func(tmp string) error {
		return &fs.PathError{Op: "open", Path: filepath.Join(tmp, "a.dll"), Err: fs.ErrNotExist}
	}
	err := s.install(t.Context(), "stardew", "gone", final, fill, func() int64 { return 0 })
	var det *DetectedError
	if usererr.KindOf(err) != usererr.Malware || !errors.As(err, &det) || !det.Removed || det.Key != "gone" || det.Game != "stardew" {
		t.Fatalf("err = %v (kind %s, detection %+v)", err, usererr.KindOf(err), det)
	}
}
