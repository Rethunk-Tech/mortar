package store

import (
	"context"
	"errors"
	"strings"
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
	_, err := s.AddArchive("stardew", p)
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
	s.AllowUnscanned("stardew", key)
	if _, err := s.AddArchive("stardew", p); err != nil {
		t.Fatalf("allowed install failed: %v", err)
	}
	if f.calls != 0 {
		t.Fatalf("the scanner ran %d times for an allowed item", f.calls)
	}
	if err := s.AddArchiveKey("stardew", "other-key", buildZip(t, map[string]string{"b": "y"})); err == nil {
		t.Fatal("the allowance covered another item")
	}
}

func TestAScannerErrorInstallsAndIsReported(t *testing.T) {
	f := &fakeScanner{err: errors.New("clamd went away")}
	s, failures := scanned(t, f)
	if _, err := s.AddArchive("stardew", buildZip(t, map[string]string{"a": "x"})); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if len(*failures) != 1 || !strings.Contains((*failures)[0].Error(), "clamd") {
		t.Fatalf("failures = %v", *failures)
	}
}

func TestNoScannerInstallsSilently(t *testing.T) {
	f := &fakeScanner{err: avscan.ErrNoScanner}
	s, failures := scanned(t, f)
	if _, err := s.AddArchive("stardew", buildZip(t, map[string]string{"a": "x"})); err != nil || len(*failures) != 0 {
		t.Fatalf("err = %v, failures = %v", err, *failures)
	}
}

func TestLoaderBundlesAreNotScanned(t *testing.T) {
	f := &fakeScanner{hit: &avscan.Detection{Name: "Heuristic"}}
	s, _ := scanned(t, f)
	dir := t.TempDir()
	if err := s.AddDir("lethal-company", LoaderKey("bepinex5", "5.4.2"), dir); err != nil {
		t.Fatalf("loader bundle was scanned: %v", err)
	}
	if f.calls != 0 {
		t.Fatalf("scanner ran %d times", f.calls)
	}
}
