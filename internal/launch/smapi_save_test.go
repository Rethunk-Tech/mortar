package launch

import "testing"

func TestLoadedSaveReadsSMAPIContextLine(t *testing.T) {
	log := "[12:00:00 INFO  SMAPI] Mods path: /mods\n" +
		"[12:01:00 INFO  SMAPI]    Context: loaded save 'Pierre_123456789'.\n"
	got, ok := LoadedSave(log)
	if !ok || got != "Pierre_123456789" {
		t.Fatalf("got %q %v", got, ok)
	}
}

func TestLoadedSaveUsesLastSMAPILine(t *testing.T) {
	log := "[12:01:00 INFO  SMAPI] Context: loaded save 'Old_1'.\n" +
		"[12:02:00 INFO  OtherMod] loaded save 'Nope_2'.\n" +
		"[12:03:00 INFO  SMAPI] loaded save 'New_3'.\n"
	got, ok := LoadedSave(log)
	if !ok || got != "New_3" {
		t.Fatalf("got %q %v", got, ok)
	}
}

func TestLoadedSaveEmptyWhenMissing(t *testing.T) {
	if _, ok := LoadedSave("[12:00:00 INFO  SMAPI] Loaded 2 mods\n"); ok {
		t.Fatal("expected no save")
	}
}
