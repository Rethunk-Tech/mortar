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
		if got, err := store.record("sdv", "run-"+strconv.Itoa(i), mod, true, nil); err != nil || len(got) != 0 {
			t.Fatalf("run %d = %v, %v", i, got, err)
		}
	}
	got, err := store.record("sdv", "run-5", mod, true, nil)
	if err != nil || len(got) != 1 || got[0].ModID != 42 {
		t.Fatalf("threshold = %v, %v", got, err)
	}
	if got, err := store.record("sdv", "run-5", mod, true, nil); err != nil || len(got) != 0 {
		t.Fatalf("duplicate run = %v, %v", got, err)
	}
	if err := store.answer("sdv", 42, false); err != nil {
		t.Fatal(err)
	}
	for i := 6; i < 10; i++ {
		if got, err := store.record("sdv", "run-"+strconv.Itoa(i), mod, true, nil); err != nil || len(got) != 0 {
			t.Fatalf("cooldown run %d = %v, %v", i, got, err)
		}
	}
	if got, err := store.record("sdv", "run-10", mod, true, nil); err != nil || len(got) != 1 {
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

func TestPromptStoreKeepsGamesApart(t *testing.T) {
	t.Parallel()
	store := newPromptStore(t)
	mod := []CleanMod{{ModID: 7, Name: "A"}}
	for i := 1; i <= cleanSessionsToAsk; i++ {
		if _, err := store.record("sdv", "run-"+strconv.Itoa(i), mod, true, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.answer("sdv", 7, true); err != nil {
		t.Fatal(err)
	}
	if got, err := store.record("lc", "run-1", mod, true, nil); err != nil || len(got) != 0 {
		t.Fatalf("other game = %v, %v", got, err)
	}
	if p := store.mods["lc:7"]; p.Clean != 1 || p.Never {
		t.Fatalf("lc progress = %+v", p)
	}
	if p := store.mods["sdv:7"]; p.Clean != cleanSessionsToAsk || !p.Never {
		t.Fatalf("sdv progress = %+v", p)
	}
}

func TestPromptStoreMigratesBareIDsPerGameAndPrunes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	old := `{"7":{"clean":4,"asked":false,"never":true,"lastRun":"old"},"9":{"clean":2,"asked":false,"never":false}}`
	if err := os.WriteFile(filepath.Join(dir, promptFileName), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := openPromptStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	mod := []CleanMod{{ModID: 7, Name: "A"}}
	for _, game := range []string{"sdv", "lc"} {
		if _, err := store.record(game, "run-1", mod, true, map[int]bool{7: true}); err != nil {
			t.Fatal(err)
		}
		if p := store.mods[game+":7"]; p.Clean != 5 || !p.Never {
			t.Fatalf("%s carried %+v", game, p)
		}
	}
	// A game-scoped entry for a mod the game no longer holds is dropped on the next write.
	store.mods["sdv:99"] = promptProgress{Clean: 3}
	if _, err := store.record("sdv", "run-2", mod, true, map[int]bool{7: true}); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.mods["sdv:99"]; ok {
		t.Fatal("uninstalled entry kept")
	}
	if _, ok := store.mods["lc:7"]; !ok {
		t.Fatal("other game's entry pruned")
	}
}
