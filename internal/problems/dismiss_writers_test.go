package problems

import (
	"errors"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

type fakeRuns struct {
	id  string
	err error
}

func (f fakeRuns) LastRunID(string, string) (string, error) {
	return f.id, f.err
}

func (f fakeRuns) LastRunSummary(string, string) (string, launch.Summary, error) {
	return f.id, launch.Summary{}, f.err
}

func openDismissSettings(t *testing.T) *settings.Store {
	t.Helper()
	st, err := settings.OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestDismissForRun(t *testing.T) {
	const gameID, profileID, runID = "g", "p", "run-2"
	kind := loadFailureKind
	currentTok := kind + "\t" + runID + "\tkeep"
	oldTok := kind + "\told-run\tgone"
	otherKindTok := runErrorKind + "\told-run\tstay"
	newTok := kind + "\t" + runID + "\trow"

	t.Run("nil Runs", func(t *testing.T) {
		svc := &Service{settings: openDismissSettings(t)}
		err := svc.dismissForRun(gameID, profileID, kind, func(string) string { return newTok })
		if err == nil || err.Error() != "no run to dismiss a row of" {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("empty LastRunID", func(t *testing.T) {
		svc := &Service{settings: openDismissSettings(t), Runs: fakeRuns{}}
		err := svc.dismissForRun(gameID, profileID, kind, func(string) string { return newTok })
		if err == nil || err.Error() != "no run to dismiss a row of" {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("LastRunID error", func(t *testing.T) {
		svc := &Service{settings: openDismissSettings(t), Runs: fakeRuns{id: runID, err: errors.New("stat")}}
		err := svc.dismissForRun(gameID, profileID, kind, func(string) string { return newTok })
		if err == nil || err.Error() != "no run to dismiss a row of" {
			t.Fatalf("got %v", err)
		}
	})

	cases := []struct {
		name string
		have []string
		want []string
	}{
		{
			name: "prunes earlier runs of the same kind, keeps current run and other kinds",
			have: []string{oldTok, currentTok, otherKindTok},
			want: []string{currentTok, otherKindTok, newTok},
		},
		{
			name: "appends once when the token is already kept",
			have: []string{oldTok, newTok, otherKindTok},
			want: []string{newTok, otherKindTok},
		},
		{
			name: "creates the dismissed map",
			have: nil,
			want: []string{newTok},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := openDismissSettings(t)
			if tc.have != nil {
				if _, err := st.Update(func(v *settings.Settings) {
					v.Dismissed = map[string][]string{dismissBucket(gameID, profileID): slices.Clone(tc.have)}
				}); err != nil {
					t.Fatal(err)
				}
			}
			svc := &Service{settings: st, Runs: fakeRuns{id: runID}}
			if err := svc.dismissForRun(gameID, profileID, kind, func(id string) string {
				return kind + "\t" + id + "\trow"
			}); err != nil {
				t.Fatal(err)
			}
			got := svc.DismissedTokens(gameID, profileID)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDismissLoadFailureAndRunError(t *testing.T) {
	const gameID, profileID, runID = "g", "p", "run-9"
	st := openDismissSettings(t)
	old := loadFailureToken("run-1", LoadFailure{Kind: "crash", Plugin: "A", Message: "boom"})
	if _, err := st.Update(func(v *settings.Settings) {
		v.Dismissed = map[string][]string{dismissBucket(gameID, profileID): {old}}
	}); err != nil {
		t.Fatal(err)
	}
	svc := &Service{settings: st, Runs: fakeRuns{id: runID}}
	ctx := t.Context()
	if err := svc.DismissLoadFailure(ctx, gameID, profileID, "crash", "A", "boom"); err != nil {
		t.Fatal(err)
	}
	if err := svc.DismissRunError(ctx, gameID, profileID, "mod.key"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		loadFailureToken(runID, LoadFailure{Kind: "crash", Plugin: "A", Message: "boom"}),
		runErrorToken(runID, "mod.key"),
	}
	if got := svc.DismissedTokens(gameID, profileID); !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDismissMissingInput(t *testing.T) {
	svc := &Service{settings: openDismissSettings(t), Runs: fakeRuns{id: "run"}}
	ctx := t.Context()
	id := mod.ID("Some.Mod")
	cases := []struct {
		name string
		call func() error
		want string
	}{
		{"DismissRunError empty key", func() error { return svc.DismissRunError(ctx, "g", "p", "") }, "missing mod key"},
		{"DismissRedundant empty kind", func() error { return svc.DismissRedundant(ctx, "g", "p", "", "k", nil) }, "missing redundant mod"},
		{"DismissRedundant empty key", func() error { return svc.DismissRedundant(ctx, "g", "p", "same", "", nil) }, "missing redundant mod"},
		{"DismissAbandonedMod empty id", func() error { return svc.DismissAbandonedMod(ctx, "g", "p", "") }, "missing mod id"},
		{"DismissListedRequirement empty id", func() error { return svc.DismissListedRequirement(ctx, "g", "p", "") }, "missing requirement id"},
		{"DismissSetting empty id", func() error { return svc.DismissSetting(ctx, "g", "p", "", "field") }, "missing setting"},
		{"DismissSetting blank field", func() error { return svc.DismissSetting(ctx, "g", "p", id, "  ") }, "missing setting"},
		{"RememberSettingChoice empty id", func() error { return svc.RememberSettingChoice(ctx, "g", "p", "", "field", "1") }, "missing setting"},
		{"RememberSettingChoice blank field", func() error { return svc.RememberSettingChoice(ctx, "g", "p", id, " \t", "1") }, "missing setting"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
			if got := svc.DismissedTokens("g", "p"); len(got) != 0 {
				t.Fatalf("wrote %q", got)
			}
		})
	}
}

func TestDismissWritersAppend(t *testing.T) {
	ctx := t.Context()
	id := mod.ID("Some.Mod")
	cases := []struct {
		name string
		call func(*Service) error
		want string
	}{
		{"DismissRedundant", func(s *Service) error {
			return s.DismissRedundant(ctx, "g", "p", "same", "k", []string{"a", "b"})
		}, redundantToken("same", "k", []string{"a", "b"})},
		{"DismissAbandonedMod", func(s *Service) error { return s.DismissAbandonedMod(ctx, "g", "p", id) }, modAdviceToken(id)},
		{"DismissListedRequirement", func(s *Service) error {
			return s.DismissListedRequirement(ctx, "g", "p", id)
		}, dismissToken("listed", id.Fold())},
		{"DismissSetting", func(s *Service) error { return s.DismissSetting(ctx, "g", "p", id, "field") }, settingToken(id, "field")},
		{"RememberSettingChoice trims field", func(s *Service) error {
			return s.RememberSettingChoice(ctx, "g", "p", id, "  field  ", "on")
		}, settingChoiceToken(id, "field", "on")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &Service{settings: openDismissSettings(t), Runs: fakeRuns{id: "run"}}
			if err := tc.call(svc); err != nil {
				t.Fatal(err)
			}
			if got := svc.DismissedTokens("g", "p"); !slices.Equal(got, []string{tc.want}) {
				t.Fatalf("got %q, want %q", got, []string{tc.want})
			}
		})
	}
}

func TestAdoptDismissedAndTokens(t *testing.T) {
	const gameID, profileID = "g", "p"
	st := openDismissSettings(t)
	bucket := dismissBucket(gameID, profileID)
	if _, err := st.Update(func(v *settings.Settings) {
		v.Dismissed = map[string][]string{bucket: {"keep", "dup"}}
	}); err != nil {
		t.Fatal(err)
	}
	svc := &Service{settings: st}

	if err := svc.AdoptDismissed(gameID, profileID, []string{"dup", "new-b", "new-a", "new-a"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"keep", "dup", "new-a", "new-b"}
	got := svc.DismissedTokens(gameID, profileID)
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	got[0] = "mutated"
	if slices.Equal(svc.DismissedTokens(gameID, profileID), got) {
		t.Fatal("DismissedTokens returned the live slice")
	}

	if err := svc.AdoptDismissed(gameID, profileID, []string{"keep", "dup"}); err != nil {
		t.Fatal(err)
	}
	if got := svc.DismissedTokens(gameID, profileID); !slices.Equal(got, want) {
		t.Fatalf("after no-op got %q, want %q", got, want)
	}
}
