package bisect

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// modFolder writes a SMAPI mod beside a test so a profile can install it.
func modFolder(t *testing.T, uniqueID string) string {
	t.Helper()
	dir := t.TempDir()
	testfs.WriteFile(t, dir, "Mod/manifest.json", `{"Name":"`+uniqueID+`","Author":"Me","UniqueID":"`+uniqueID+`","Version":"1.0.0"}`)
	return dir
}

// A check whose launches cannot start (no game installed) ends in StateFailed with the launch's error, and every
// mod of the duplicate profile is switched back on before it gives up.
func TestACheckThatCannotLaunchReportsFailed(t *testing.T) {
	testfs.DataHome(t)
	settings, profiles := testenv.Env(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	if _, err := profiles.InstallLocalFolder(t.Context(), "stardew", p.ID, modFolder(t, "Me.One"), true); err != nil {
		t.Fatal(err)
	}
	s := NewService(profiles, launchsvc.NewService(t.TempDir(), settings, profiles))
	id, err := s.Start(t.Context(), "stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		status, err := s.Status(id)
		if err != nil {
			t.Fatal(err)
		}
		if status.State == StateFailed {
			if status.Error == "" || status.Result != nil || status.ModsLeft != 0 {
				t.Fatalf("failed status = %+v", status)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("state = %q, want %q", status.State, StateFailed)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := s.Stop(id); err != nil {
		t.Errorf("Stop of a finished check: %v", err)
	}
}

// SwitchOff switches the mod off in the real profile, found by its key, and refuses an empty identity.
func TestSwitchOffDisablesTheModInTheRealProfile(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Env(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	res, err := profiles.InstallLocalFolder(t.Context(), "stardew", p.ID, modFolder(t, "Me.Two"), true)
	if err != nil {
		t.Fatal(err)
	}
	entry := res.Profile.Entries[0]
	s := NewService(profiles, nil)
	if err := s.SwitchOff("stardew", p.ID, entry.Key, entry.Mods[0].ID); err != nil {
		t.Fatal(err)
	}
	mods, err := profiles.UserMods("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Enabled {
		t.Fatalf("mods after SwitchOff = %+v", mods)
	}
}

// Stop cancels the check it names, which is what ends its temporary launch.
func TestStopCancelsTheNamedCheck(t *testing.T) {
	s := NewService(nil, nil)
	cancelled := false
	j := &job{id: "bisect-1", cancel: func() { cancelled = true }}
	s.jobs = map[string]*job{j.id: j}
	if err := s.Stop(j.id); err != nil {
		t.Fatal(err)
	}
	if !cancelled {
		t.Fatal("Stop did not cancel the check")
	}
}

// modID names a mod by its catalog id, or by its entry key when it has none.
func TestModIDFallsBackToTheEntryKey(t *testing.T) {
	if got := modID(profile.Mod{ID: "mymod:Me.Thing", Key: "key"}); got != "mymod:Me.Thing" {
		t.Errorf("modID with an id = %q", got)
	}
	if got := modID(profile.Mod{Key: "Folder"}); got != "Folder" {
		t.Errorf("modID without an id = %q", got)
	}
}

// One package's group is found whole: a framework switched with its pack never leaves the other half convicted alone.
func TestFindFindsAGroupTogether(t *testing.T) {
	mods := []Mod{
		{ID: "pack", Group: "g"},
		{ID: "framework", Group: "g"},
		{ID: "other"},
	}
	got, err := find(t.Context(), mods, func(_ context.Context, off []string) (bool, error) {
		return !slices.Contains(off, "pack"), nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(got))
	for _, m := range got {
		ids = append(ids, m.ID)
	}
	if !slices.Contains(ids, "pack") || !slices.Contains(ids, "framework") || slices.Contains(ids, "other") {
		t.Fatalf("found %v, want the group without the clean mod", ids)
	}
}

// A launch failure ends the check at once with that error, never as "no mod is to blame".
func TestFindReturnsTheRunnerError(t *testing.T) {
	boom := errors.New("boom")
	_, err := find(t.Context(), []Mod{{ID: "a"}}, func(context.Context, []string) (bool, error) {
		return false, boom
	}, nil)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the runner's", err)
	}
}

// A check with nothing to test names that, and a cancelled context stops before the next run.
func TestFindRefusesNoModsAndStopsWhenCancelled(t *testing.T) {
	if _, err := find(t.Context(), nil, func(context.Context, []string) (bool, error) { return true, nil }, nil); err == nil {
		t.Error("no mods to test was accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := find(ctx, []Mod{{ID: "a"}, {ID: "b"}}, func(context.Context, []string) (bool, error) { return true, nil }, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// Progress is reported once per step, counting down the suspects, so the window can show how far the check is.
func TestFindReportsProgressEachStep(t *testing.T) {
	var steps []Progress
	mods := []Mod{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	if _, err := find(t.Context(), mods, func(_ context.Context, off []string) (bool, error) {
		return len(off) == 0 || slices.Contains(off, "c"), nil
	}, func(p Progress) { steps = append(steps, p) }); err != nil {
		t.Fatal(err)
	}
	if len(steps) == 0 || steps[0].Step != 1 || steps[0].Total < 1 || steps[0].ModsLeft != len(mods) {
		t.Fatalf("steps = %+v", steps)
	}
	for i, p := range steps {
		if p.Step != i+1 || p.ModsLeft > len(mods) {
			t.Fatalf("step %d = %+v", i, p)
		}
	}
}

func TestStatusOfAKnownCheckCopiesItsResult(t *testing.T) {
	s := NewService(nil, nil)
	j := &job{id: "bisect-7", game: "stardew", profile: "p", cancel: func() {}}
	s.jobs = map[string]*job{j.id: j}
	s.setStatus(Status{ID: j.id, State: StateDone, Result: &Result{Mods: []ModResult{{Key: "k", Name: "N"}}}})
	first, err := s.Status(j.id)
	if err != nil {
		t.Fatal(err)
	}
	first.Result.Mods[0].Name = "changed"
	second, err := s.Status(j.id)
	if err != nil {
		t.Fatal(err)
	}
	if second.Result.Mods[0].Name != "N" {
		t.Fatalf("a caller's edit reached the stored result: %+v", second.Result)
	}
	if !strings.Contains(func() string {
		_, err := s.Status("bisect-none")
		return err.Error()
	}(), "bisect-none") {
		t.Error("an unknown check's error does not name it")
	}
}
