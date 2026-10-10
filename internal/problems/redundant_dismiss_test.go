package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func TestDismissedRedundantRowReturnsWhenItsPartnersChange(t *testing.T) {
	row := func(key string, by ...string) framework.Redundant {
		r := framework.Redundant{Kind: "sameJob", Key: key, Name: key}
		for _, b := range by {
			r.By = append(r.By, framework.ModRef{Key: b, Name: b})
		}
		return r
	}
	tokens := []string{redundantToken("sameJob", "walkie", []string{"things"})}
	kept, dismissed := hideDismissedRedundant([]framework.Redundant{row("walkie", "things"), row("things", "walkie")}, tokens)
	if len(kept) != 1 || kept[0].Key != "things" || len(dismissed) != 1 || dismissed[0].Redundant.Key != "walkie" {
		t.Fatalf("kept %+v, dismissed %+v", kept, dismissed)
	}
	kept, dismissed = hideDismissedRedundant([]framework.Redundant{row("walkie", "things", "other")}, tokens)
	if len(kept) != 1 || len(dismissed) != 0 {
		t.Fatalf("a new partner should bring the row back: kept %+v, dismissed %+v", kept, dismissed)
	}
	if redundantToken("sameJob", "a", []string{"y", "x"}) != redundantToken("sameJob", "a", []string{"x", "y"}) {
		t.Fatal("the token depends on the order of the partners")
	}
}
