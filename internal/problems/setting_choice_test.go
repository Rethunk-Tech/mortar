package problems

import "testing"

func TestRememberedSettingChoiceHidesHintUntilValueChanges(t *testing.T) {
	setting := SettingHint{
		UniqueID: "Cornucopia.MoreCrops",
		Field:    "Enable Extended Trees Pack",
		Current:  "false",
	}
	tokens := []string{dismissToken(
		"setting-choice",
		"cornucopia.morecrops\tenable extended trees pack\tfalse",
	)}

	if got := hideDismissedSettings([]SettingHint{setting}, tokens); len(got) != 0 {
		t.Fatalf("remembered setting choice left reverse hint: %+v", got)
	}

	setting.Current = "true"
	got := hideDismissedSettings([]SettingHint{setting}, tokens)
	if len(got) != 1 {
		t.Fatalf("changed setting value hid hint: %+v", got)
	}
}
