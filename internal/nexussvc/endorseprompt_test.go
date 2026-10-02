package nexussvc

import (
	"path/filepath"
	"strconv"
	"testing"
)

func newPromptStore(t *testing.T) *promptStore {
	t.Helper()
	return &promptStore{path: filepath.Join(t.TempDir(), promptFileName), mods: map[string]promptProgress{}}
}

func TestPromptStoreCountsCleanRunsAndRetriesLater(t *testing.T) {
	store := newPromptStore(t)
	mod := []CleanMod{{ModID: 42, Name: "A", Version: "1.0"}}

	for i := 1; i < cleanSessionsToAsk; i++ {
		if got, err := store.record("run-"+strconv.Itoa(i), mod, true); err != nil || len(got) != 0 {
			t.Fatalf("run %d = %v, %v", i, got, err)
		}
	}
	got, err := store.record("run-5", mod, true)
	if err != nil || len(got) != 1 || got[0].ModID != 42 {
		t.Fatalf("threshold = %v, %v", got, err)
	}
	if got, err := store.record("run-5", mod, true); err != nil || len(got) != 0 {
		t.Fatalf("duplicate run = %v, %v", got, err)
	}
	if err := store.answer(42, false); err != nil {
		t.Fatal(err)
	}
	for i := 6; i < 10; i++ {
		if got, err := store.record("run-"+strconv.Itoa(i), mod, true); err != nil || len(got) != 0 {
			t.Fatalf("cooldown run %d = %v, %v", i, got, err)
		}
	}
	if got, err := store.record("run-10", mod, true); err != nil || len(got) != 1 {
		t.Fatalf("retry threshold = %v, %v", got, err)
	}
}

func TestPromptStoreEligibility(t *testing.T) {
	store := newPromptStore(t)
	mods := []CleanMod{
		{ModID: 1, Name: "undecided"},
		{ModID: 2, Name: "endorsed", Endorsement: "Endorsed"},
		{ModID: 3, Name: "abstained", Endorsement: "Abstained"},
	}
	for i := range cleanSessionsToAsk {
		got, err := store.record("run-"+strconv.Itoa(i), mods, i == cleanSessionsToAsk-1)
		if err != nil {
			t.Fatal(err)
		}
		if i < cleanSessionsToAsk-1 && len(got) != 0 {
			t.Fatalf("early prompts = %v", got)
		}
		if i == cleanSessionsToAsk-1 {
			if len(got) != 1 || got[0].ModID != 1 {
				t.Fatalf("eligible prompts = %v", got)
			}
		}
	}
}
