package queue

import (
	"context"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestNeedsRootAnswer(t *testing.T) {
	t.Parallel()
	ask := &profile.RemapAsk{Key: "store-key", Source: profile.Source{Kind: profile.KindNexus, ModID: 1, FileID: 2}}
	s := &Service{
		d: Deps{
			Dir: t.TempDir(),
			Now: func() time.Time { return time.Unix(1, 0).UTC() },
		},
		kick:    make(chan struct{}, 1),
		cancels: map[string]context.CancelFunc{},
	}
	s.items = []*Item{{
		ID: "a", Game: "stardew", Profile: "p", State: StateNeedsRoot, staged: "store-key", Remap: ask,
	}}

	s.AnswerRoot("a", "ModFolder")
	if it := s.find("a"); it.State != StateQueued || it.chosenRoot != "ModFolder" || it.staged != "store-key" {
		t.Fatalf("after AnswerRoot: %+v", it)
	}

}

// A chosen root installs from the store even when the item's archive is still on disk, and a Nexus or package item
// keeps its own source instead of becoming a GitHub one.
func TestInstallStagedHonoursChosenRootAndSource(t *testing.T) {
	t.Parallel()
	var remapKey string
	var staged profile.Source
	s := &Service{
		d: Deps{
			Dir: t.TempDir(),
			Now: func() time.Time { return time.Unix(1, 0).UTC() },
			InstallRemap: func(_, _, key, _ string, _ profile.Source) (profile.InstallResult, error) {
				remapKey = key
				return profile.InstallResult{}, nil
			},
			InstallStaged: func(_, _, _ string, src profile.Source) (profile.InstallResult, error) {
				staged = src
				return profile.InstallResult{}, nil
			},
		},
		kick:    make(chan struct{}, 1),
		cancels: map[string]context.CancelFunc{},
	}
	s.items = []*Item{{ID: "a", ModID: 1, FileID: 2}, {ID: "b", ModID: 7, FileID: 8}, {ID: "c", Package: "A-B", Source: profile.KindVSModDB}}
	nx := Item{ID: "a", Game: "stardew", ModID: 1, FileID: 2, staged: "k", readyZip: true, chosenRoot: "Mods"}
	if err := s.installStaged(context.Background(), nx); err != nil || remapKey != "k" {
		t.Fatalf("chosen root ignored: key %q, %v", remapKey, err)
	}
	nx.chosenRoot = ""
	nx.ID, nx.ModID = "b", 7
	nx.readyZip = false
	if err := s.installStaged(context.Background(), nx); err != nil || staged.Kind != profile.KindNexus || staged.ModID != 7 {
		t.Fatalf("nexus source: %+v %v", staged, err)
	}
	pk := Item{ID: "c", Game: "vintagestory", Package: "A-B", Source: profile.KindVSModDB, staged: "k"}
	if err := s.installStaged(context.Background(), pk); err != nil || staged.Kind != profile.KindVSModDB {
		t.Fatalf("package source: %+v %v", staged, err)
	}
}

// A package paused for a question installs from its staged copy instead of downloading again.
func TestNextInstallsStagedPackage(t *testing.T) {
	t.Parallel()
	s := &Service{d: Deps{Premium: func() bool { return true }}}
	s.items = []*Item{{ID: "p", Package: "A-B", State: StateQueued, staged: "k"}}
	if it, act, _ := s.next(nil); it == nil || act != install {
		t.Fatalf("next = %v %v, want install", it, act)
	}
}

// A stored install of an item cancelled meanwhile neither installs nor revives it.
func TestInstallStoredSkipsACancelledItem(t *testing.T) {
	t.Parallel()
	called := false
	s := &Service{d: Deps{
		InstallStaged: func(_, _, _ string, _ profile.Source) (profile.InstallResult, error) {
			called = true
			return profile.InstallResult{}, nil
		},
	}}
	s.items = []*Item{{ID: "a", ModID: 1, FileID: 2, Kind: KindUpdate, State: StateCancelled}}
	done, err := s.installStored(context.Background(), nil, *s.items[0], nexus.Mod{})
	if !done || err == nil || called || s.items[0].State != StateCancelled {
		t.Fatalf("done %v err %v called %v state %s", done, err, called, s.items[0].State)
	}
}
