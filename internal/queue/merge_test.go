package queue

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestNexusOptionalFileOffersMergeIntoTheSamePageEntry(t *testing.T) {
	f := newFixture(t)
	f.samePage = func(_, _ string, modID, fileID int, category string) (profile.MergeAsk, bool) {
		if modID != 1 || fileID != 10 || category != "MAIN" {
			t.Errorf("same page args %d %d %q", modID, fileID, category)
		}
		return profile.MergeAsk{EntryKey: "nexus-1-9", Label: "Alpha", DefaultAdd: true}, true
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
	f.samePage = func(_, _ string, _, _ int, _ string) (profile.MergeAsk, bool) {
		return profile.MergeAsk{EntryKey: "nexus-1-9", Label: "Alpha", DefaultAdd: false}, true
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
