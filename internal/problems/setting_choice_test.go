package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestRememberedSettingChoiceHidesHintUntilValueChanges(t *testing.T) {
	setting := framework.SettingHint{
		ID:      "smapi:Cornucopia.MoreCrops",
		Field:   "Enable Extended Trees Pack",
		Current: "false",
	}
	tokens := []string{dismissToken(
		"setting-choice",
		"smapi:cornucopia.morecrops\tenable extended trees pack\tfalse",
	)}

	if got, dismissed := hideDismissedSettings([]framework.SettingHint{setting}, tokens); len(got) != 0 || len(dismissed) != 1 {
		t.Fatalf("remembered setting choice left reverse hint: %+v", got)
	}

	setting.Current = "true"
	got, _ := hideDismissedSettings([]framework.SettingHint{setting}, tokens)
	if len(got) != 1 {
		t.Fatalf("changed setting value hid hint: %+v", got)
	}
}
