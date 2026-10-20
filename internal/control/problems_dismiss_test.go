package control

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

type oneRun string

func (r oneRun) LastRunID(string, string) (string, error) { return string(r), nil }

func (oneRun) LastRunSummary(string, string) (string, launch.Summary, error) {
	return "", launch.Summary{}, nil
}

func TestDismissingALoadFailureRowByIndexHidesItUntilTheNextRun(t *testing.T) {
	s, _ := servicesWithItems(t)
	prof, err := s.Profiles.Create("lethal-company", "Pack")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.Store.ProfileDir("lethal-company", prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "BepInEx", "LogOutput.log")
	if err := os.MkdirAll(filepath.Dir(log), 0o750); err != nil {
		t.Fatal(err)
	}
	line := "[Error  :   BepInEx] Failed to load patcher [Fixture.dll] boom\n"
	if err := os.WriteFile(log, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	s.Problems.Runs = oneRun("run-1")
	rows := func() int {
		res, err := s.Problems.Problems(t.Context(), "lethal-company", prof.ID)
		if err != nil {
			t.Fatal(err)
		}
		return len(res.LoadFailures)
	}
	if rows() != 1 {
		t.Fatalf("the log's failure is not listed: %d rows", rows())
	}

	if err := s.dismissProblem(t.Context(), "lethal-company", prof.ID, 0); err == nil {
		t.Fatal("index 0 was accepted")
	}
	if err := s.dismissProblem(t.Context(), "lethal-company", prof.ID, 9); err == nil {
		t.Fatal("an index past the dismissable rows was accepted")
	}
	if err := s.dismissProblem(t.Context(), "lethal-company", prof.ID, 1); err != nil {
		t.Fatal(err)
	}
	if rows() != 0 {
		t.Fatal("the dismissed failure is still listed")
	}
	res, err := s.Problems.Problems(t.Context(), "lethal-company", prof.ID)
	if err != nil || len(res.Dismissed) != 1 {
		t.Fatalf("dismissed = %+v, %v", res.Dismissed, err)
	}
	if err := s.restoreDismissedProblem(t.Context(), "lethal-company", prof.ID, "", 1); err != nil {
		t.Fatal(err)
	}
	if rows() != 1 {
		t.Fatal("restoring by index did not bring the failure back")
	}

	if err := s.dismissProblem(t.Context(), "lethal-company", prof.ID, 1); err != nil {
		t.Fatal(err)
	}
	s.Problems.Runs = oneRun("run-2")
	if rows() != 1 {
		t.Fatal("a dismissal outlived the run it was made for")
	}
}
