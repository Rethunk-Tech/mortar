package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestHideDismissedSoftOnly(t *testing.T) {
	in := []framework.AssetConflict{
		{Kind: "load", Target: "a"},
		{Kind: "edit", Target: "b"},
	}
	got, dismissed := hideDismissed(in, []string{dismissToken("edit", "b"), dismissToken("load", "a")})
	if len(got) != 0 || len(dismissed) != 2 {
		t.Fatalf("got %+v dismissed %+v", got, dismissed)
	}
}
