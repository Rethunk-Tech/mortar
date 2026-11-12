package contentpatcher

import (
	"testing"

	cp "github.com/Rethunk-Tech/go-contentpatcher"
)

func TestFrameworkHintExplainsNoEffectSetting(t *testing.T) {
	got := frameworkHint(cp.Hint{BecausePack: "Cave", DeadTarget: "maps/farmcave"}).Description
	if want := "This setting has no effect because Cave loads maps/farmcave over it."; got != want {
		t.Fatalf("Description = %q, want %q", got, want)
	}
}
