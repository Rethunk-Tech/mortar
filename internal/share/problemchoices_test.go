package share

import (
	"errors"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestProblemChoicesRoundTripAndAreDroppedWhenNotIncluded(t *testing.T) {
	win, lose := mod.SMAPI("Me.Win"), mod.SMAPI("Me.Lose")
	p := profile.Profile{Name: "P", Entries: []profile.Entry{{
		Key: "k", Source: profile.Source{Kind: profile.KindNexus, ModID: 7, FileID: 70},
		Mods: []profile.Component{{ID: win, Folder: ".", LoadAfter: []mod.ID{lose}}},
	}}}
	inc := OwnInclude()
	inc.Dismissed = []string{"broken\tme.old", "no tab", strings.Repeat("x\t", 400)}
	pv := writeRead(t, "stardew", p, t.TempDir(), inc)
	if len(pv.Choices.Dismissed) != 1 || pv.Choices.Dismissed[0] != "broken\tme.old" {
		t.Fatalf("dismissed = %q, want only the well-formed token", pv.Choices.Dismissed)
	}
	if len(pv.Choices.Wins) != 1 || pv.Choices.Wins[0].Winner != win || pv.Choices.Wins[0].Loser != lose || pv.Choices.Wins[0].Ref.FileID != 70 {
		t.Fatalf("wins = %+v", pv.Choices.Wins)
	}
	inc.ProblemChoices = false
	if pv := writeRead(t, "stardew", p, t.TempDir(), inc); !pv.Choices.Empty() {
		t.Fatalf("choices = %+v; want none when not included", pv.Choices)
	}
}

func TestProblemChoicesWithAnUnsafeTokenAreRefused(t *testing.T) {
	err := checkProblemChoices(ProblemChoices{Dismissed: []string{"a\tb\x00"}})
	if !errors.Is(err, ErrBadFile) {
		t.Fatalf("err = %v", err)
	}
}
