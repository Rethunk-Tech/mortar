package nexussvc

import (
	"os"
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
	// Only the undecided mod is ever asked about: the others are already endorsed or abstained.
	mod := []CleanMod{
		{ModID: 42, Name: "A", Version: "1.0"},
		{ModID: 2, Name: "endorsed", Endorsement: "Endorsed"},
		{ModID: 3, Name: "abstained", Endorsement: "Abstained"},
	}

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

func TestOpenPromptStoreMovesCorruptFileAside(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, promptFileName), []byte("[1,"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := openPromptStore(dir)
	if err != nil || len(p.mods) != 0 {
		t.Fatalf("p=%v err=%v", p, err)
	}
}
