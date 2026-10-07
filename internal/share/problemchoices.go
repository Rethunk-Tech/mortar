package share

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Caps on the problem choices a .mortar file carries, so a file cannot grow its receiver's settings without bound.
const (
	maxDismissed      = 1000
	maxDismissedBytes = 300
	maxWins           = 1000
)

// ProblemChoices are the decisions a player made about their profile's problems: which problems they dismissed
// (also the setting hints they settled), and which pack they made win an edit conflict.
type ProblemChoices struct {
	// Dismissed are problem tokens, as the problem checker records them for a profile.
	Dismissed []string `json:"dismissed,omitempty"`
	// Wins say a pack loads after the one it overrides.
	Wins []Win `json:"wins,omitempty"`
}

// Win records that Winner, a pack of the download Ref names, loads after Loser.
type Win struct {
	Ref    Ref    `json:"ref"`
	Winner mod.ID `json:"winner"`
	Loser  mod.ID `json:"loser"`
}

// Empty reports whether there is nothing to apply.
func (c ProblemChoices) Empty() bool { return len(c.Dismissed) == 0 && len(c.Wins) == 0 }

// Count is how many decisions there are.
func (c ProblemChoices) Count() int { return len(c.Dismissed) + len(c.Wins) }

// collectProblemChoices gathers the wins the shared entries hold plus the dismissals the caller read from the
// player's settings. keep says which entries the file carries.
func collectProblemChoices(p profile.Profile, dismissed []string, keep func(profile.Entry) bool) *ProblemChoices {
	var c ProblemChoices
	for _, t := range dismissed {
		if validToken(t) && len(c.Dismissed) < maxDismissed {
			c.Dismissed = append(c.Dismissed, t)
		}
	}
	for _, e := range p.Entries {
		if e.Source.Bundled() || e.IsOverlay() || !keep(e) {
			continue
		}
		id, ok := identityOf(e)
		if !ok {
			continue
		}
		for _, m := range e.Mods {
			for _, loser := range m.LoadAfter {
				if validID(m.ID) && validID(loser) && len(c.Wins) < maxWins {
					c.Wins = append(c.Wins, Win{Ref: id, Winner: m.ID, Loser: loser})
				}
			}
		}
	}
	if c.Empty() {
		return nil
	}
	return &c
}

func validToken(t string) bool {
	return len(t) <= maxDismissedBytes && strings.Contains(t, "\t") &&
		!strings.ContainsFunc(t, func(r rune) bool { return r != '\t' && unicode.IsControl(r) })
}

func checkProblemChoices(c ProblemChoices) error {
	if len(c.Dismissed) > maxDismissed || len(c.Wins) > maxWins {
		return fmt.Errorf("%w: too many problem choices", ErrBadFile)
	}
	for _, t := range c.Dismissed {
		if !validToken(t) {
			return fmt.Errorf("%w: bad dismissed problem", ErrBadFile)
		}
	}
	for _, w := range c.Wins {
		if !validID(w.Winner) || !validID(w.Loser) || !w.Ref.valid() {
			return fmt.Errorf("%w: bad conflict winner", ErrBadFile)
		}
	}
	return nil
}
