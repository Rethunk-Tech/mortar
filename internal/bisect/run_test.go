package bisect

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/mod"
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
			// The launch's own failure, not "no mod is to blame": reporting the latter would hide that the game is missing.
			if !strings.Contains(status.Error, "not installed") || strings.Contains(status.Error, "did not crash") {
				t.Fatalf("failed status = %+v, want the missing install named", status)
			}
			if status.Result != nil || status.ModsLeft != 0 {
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
	// An empty key still names the mod when its id is given: the first entry holding it is switched off.
	if err := s.SwitchOff("stardew", p.ID, "", entry.Mods[0].ID); err != nil {
		t.Fatal(err)
	}
	mods, err := profiles.UserMods("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Enabled {
		t.Fatalf("mods after SwitchOff = %+v", mods)
	}
	// Neither names the mod, so the guard refuses rather than letting the store answer for some entry.
	err = s.SwitchOff("stardew", p.ID, "", "")
	if err == nil || !strings.Contains(err.Error(), "identity is required") {
		t.Fatalf("SwitchOff with no identity = %v, want the guard's refusal", err)
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

// A content pack's graph entry depends on the framework it is written for as well as what its manifest needs, since
// switching that framework off takes the pack with it.
func TestCheckModsAddsTheContentPackFramework(t *testing.T) {
	got := checkMods([]profile.Mod{
		{Key: "Pack-1", ID: "smapi:Pack", Needs: []mod.ID{"smapi:Needed"}, ContentPackFor: "smapi:Framework"},
		{Key: "Plain-1", ID: "smapi:Plain"},
	})
	want := []Mod{
		{ID: "smapi:Pack", Dependencies: []string{"smapi:Needed", "smapi:Framework"}, Group: "Pack-1"},
		{ID: "smapi:Plain", Dependencies: []string{}, Group: "Plain-1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("checkMods = %#v, want %#v", got, want)
	}
}

// A step switches on every mod of the check before switching off the ones it names, so a mod a previous step
// cleared is on again for this launch.
func TestSetCheckModsEnablesAllThenDisablesTheNamed(t *testing.T) {
	testfs.DataHome(t)
	_, profiles := testenv.Env(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	for _, uniqueID := range []string{"Me.One", "Me.Two"} {
		if _, err := profiles.InstallLocalFolder(t.Context(), "stardew", p.ID, modFolder(t, uniqueID), true); err != nil {
			t.Fatal(err)
		}
	}
	userMods, err := profiles.UserMods("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	j := &job{game: "stardew", tempID: p.ID, mods: userMods}
	for _, pm := range userMods {
		j.refs = append(j.refs, profile.EnableRef{Key: pm.Key, ID: pm.ID})
	}
	s := NewService(profiles, nil)
	if err := s.setCheckMods(j, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.setCheckMods(j, []string{modID(userMods[0])}); err != nil {
		t.Fatal(err)
	}
	mods, err := profiles.UserMods("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	enabled := map[mod.ID]bool{}
	for _, m := range mods {
		enabled[m.ID] = m.Enabled
	}
	if len(enabled) != 2 || enabled[userMods[0].ID] || !enabled[userMods[1].ID] {
		t.Fatalf("after the step = %v, want %s off and %s on", enabled, userMods[0].ID, userMods[1].ID)
	}
}

// A launch failure ends the check at once with that error, never as "no mod is to blame"; one package's group is
// found whole, a framework switched with its pack never leaving the other half convicted alone.
func TestFindNamesAGroupAndReturnsTheRunnerError(t *testing.T) {
	mods := []Mod{{ID: "pack", Group: "g"}, {ID: "framework", Group: "g"}, {ID: "other"}}
	got, err := find(t.Context(), mods, func(_ context.Context, off []string) (bool, error) {
		return !slices.Contains(off, "pack"), nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range got {
		ids = append(ids, m.ID)
	}
	if !slices.Contains(ids, "pack") || !slices.Contains(ids, "framework") || slices.Contains(ids, "other") {
		t.Fatalf("found %v, want the group without the clean mod", ids)
	}
	boom := errors.New("boom")
	if _, err := find(t.Context(), []Mod{{ID: "a"}}, func(context.Context, []string) (bool, error) {
		return false, boom
	}, nil); !errors.Is(err, boom) {
		t.Fatalf("a failing launch: err = %v, want the runner's", err)
	}
}

// A check with nothing to test names that, and a cancelled context stops before the next run; a launch that fails
// ends the check at once with that error, never as "no mod is to blame". A cancel that lands on the step narrowing
// the suspects to one finishes the check instead, since that mod is already the answer.
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
	boom := errors.New("boom")
	_, err = find(t.Context(), []Mod{{ID: "a"}}, func(context.Context, []string) (bool, error) {
		return false, boom
	}, nil)
	if !errors.Is(err, boom) {
		t.Fatalf("a failing launch: err = %v, want the runner's", err)
	}

	lastCtx, stopLast := context.WithCancel(t.Context())
	defer stopLast()
	got, err := find(lastCtx, []Mod{{ID: "a"}, {ID: "b"}}, func(_ context.Context, off []string) (bool, error) {
		if len(off) == 0 {
			return true, nil
		}
		stopLast()
		return false, nil
	}, nil)
	if err != nil || len(got) != 1 || got[0].ID != "a" {
		t.Fatalf("found %v, %v; want a with no error", got, err)
	}
}

// Progress is reported once per step, counting down the suspects, so the window can show how far the check is: the
// step count is the number of halvings it will take, not one per suspect.
func TestFindReportsProgressEachStep(t *testing.T) {
	var steps []Progress
	mods := []Mod{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	if _, err := find(t.Context(), mods, func(_ context.Context, off []string) (bool, error) {
		return len(off) == 0 || slices.Contains(off, "c"), nil
	}, func(p Progress) { steps = append(steps, p) }); err != nil {
		t.Fatal(err)
	}
	want := []Progress{{Step: 1, Total: 2, ModsLeft: 4}, {Step: 2, Total: 2, ModsLeft: 2}}
	if !slices.Equal(steps, want) {
		t.Fatalf("steps = %+v, want %+v", steps, want)
	}
}

// A cancelled check stops rather than failing: Stop cancels the job it names, and Status returns a copy so a caller
// editing the result does not change what the next reader sees.
func TestStopCancelsAndStatusCopiesTheResult(t *testing.T) {
	s := NewService(nil, nil)
	cancelled := false
	j := &job{id: "bisect-7", game: "stardew", profile: "p", cancel: func() { cancelled = true }}
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
	if err := s.Stop(j.id); err != nil {
		t.Fatal(err)
	}
	if !cancelled {
		t.Error("Stop did not cancel the check")
	}
	if !strings.Contains(func() string {
		_, err := s.Status("bisect-none")
		return err.Error()
	}(), "bisect-none") {
		t.Error("an unknown check's error does not name it")
	}
}
