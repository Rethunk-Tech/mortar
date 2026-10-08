package settings

import "testing"

func TestAskEndorseModsDefaultsOffAndCanBeEnabled(t *testing.T) {
	store, _ := open(t)
	if store.Get().AskEndorseMods == nil || *store.Get().AskEndorseMods {
		t.Fatal("ask endorse mods default is on")
	}
	if err := NewService(store).SetAskEndorseMods(true); err != nil {
		t.Fatal(err)
	}
	if got := store.Get().AskEndorseMods; got == nil || !*got {
		t.Fatalf("ask endorse mods = %v", got)
	}
}
