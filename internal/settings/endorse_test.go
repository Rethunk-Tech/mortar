package settings

import "testing"

func TestAskEndorseModsDefaultsOnAndCanBeDisabled(t *testing.T) {
	store, _ := open(t)
	if store.Get().AskEndorseMods == nil || !*store.Get().AskEndorseMods {
		t.Fatal("ask endorse mods default is off")
	}
	if err := NewService(store).SetAskEndorseMods(false); err != nil {
		t.Fatal(err)
	}
	if got := store.Get().AskEndorseMods; got == nil || *got {
		t.Fatalf("ask endorse mods = %v", got)
	}
}
