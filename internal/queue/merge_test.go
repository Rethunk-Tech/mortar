package queue

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestNexusOptionalFileOffersMergeIntoTheSamePageEntry(t *testing.T) {
	f := newFixture(t)
	f.samePage = func(_, _ string, in profile.IncomingFile) (profile.MergeAsk, int, bool) {
		if in.ModID != 1 || in.FileID != 10 || in.Category != "MAIN" {
			t.Errorf("same page args %+v", in)
		}
		return profile.MergeAsk{EntryKey: "nexus-1-9", Label: "Alpha", DefaultAdd: true}, 0, true
	}
	var extras []string
	f.installExtra = func(_, _, entryKey, path string, src profile.Source) (profile.InstallResult, error) {
		b, err := fsx.ReadFile(path)
		if err != nil || string(b) != payload || entryKey != "nexus-1-9" || src.ModID != 1 {
			t.Errorf("extra install %s %q %+v %v", entryKey, b, src, err)
		}
		extras = append(extras, entryKey)
		return profile.InstallResult{}, nil
	}
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{req(10)}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("merge choice", f.item(StateNeedsMerge))
	if st.Items[0].Merge == nil || st.Items[0].Merge.EntryKey != "nexus-1-9" || !st.Items[0].Merge.DefaultAdd {
		t.Fatalf("ask %+v", st.Items[0].Merge)
	}
	f.s.AnswerMerge(st.Items[0].ID, true)
	f.wait("merged", f.item(StateDone))
	if len(extras) != 1 || len(f.installs) != 0 {
		t.Fatalf("extra %v installs %d", extras, len(f.installs))
	}
	f.leftovers()
}

func TestNexusSecondMainFileCanStayASeparateEntry(t *testing.T) {
	f := newFixture(t)
	f.samePage = func(_, _ string, _ profile.IncomingFile) (profile.MergeAsk, int, bool) {
		return profile.MergeAsk{EntryKey: "nexus-1-9", Label: "Alpha", DefaultAdd: false}, 0, true
	}
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{req(10)}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("merge choice", f.item(StateNeedsMerge))
	f.s.AnswerMerge(st.Items[0].ID, false)
	f.wait("separate", f.item(StateDone))
	if len(f.installs) != 1 {
		t.Fatalf("installs %d", len(f.installs))
	}
	f.leftovers()
}

func TestNexusUpdateFromTheSamePageInstallsInPlaceOfTheEntry(t *testing.T) {
	f := newFixture(t)
	f.samePage = func(_, _ string, in profile.IncomingFile) (profile.MergeAsk, int, bool) {
		if len(in.Files) == 0 {
			return profile.MergeAsk{EntryKey: "nexus-1-9", Label: "Alpha"}, 0, true
		}
		return profile.MergeAsk{}, 9, false
	}
	f.start()
	if _, err := f.s.Add(t.Context(), []Request{req(10)}); err != nil {
		t.Fatal(err)
	}
	st := f.wait("updated", f.item(StateDone))
	if st.Items[0].Current != 9 || len(f.installs) != 1 {
		t.Fatalf("current %d installs %d", st.Items[0].Current, len(f.installs))
	}
	f.leftovers()
}

func TestAWantJoiningAFailedItemTakesItsBatch(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.Add(t.Context(), []Request{req(10)}); err != nil {
		t.Fatal(err)
	}
	f.s.mu.Lock()
	f.s.items[0].State = StateFailed
	f.s.mu.Unlock()
	again := req(10)
	again.BatchID = "update-all"
	items, err := f.s.Add(t.Context(), []Request{again})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].BatchID != "update-all" || items[0].State != StateQueued {
		t.Fatalf("joined item = batch %q, state %s", items[0].BatchID, items[0].State)
	}
}
