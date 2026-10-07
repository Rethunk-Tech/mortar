package sharesvc

import (
	"log"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

// Dismissals is where a profile's dismissed problems are kept.
type Dismissals interface {
	DismissedTokens(game, profileID string) []string
	AdoptDismissed(game, profileID string, tokens []string) error
}

// applySharedChoices adds a share's problem choices to the profile. Dismissals are added to the profile's own. A
// win applies once the download it names is on the profile and both packs are there; the profile's own choices
// stay, so a win the profile already reversed is not applied. Wins whose mods are missing are skipped, and a later
// call applies them once the mods land.
func (s *Service) applySharedChoices(game, profileID string, c share.ProblemChoices) error {
	if c.Empty() {
		return nil
	}
	if len(c.Dismissed) > 0 && s.d.Dismissals != nil {
		if err := s.d.Dismissals.AdoptDismissed(game, profileID, c.Dismissed); err != nil {
			return err
		}
	}
	for _, w := range c.Wins {
		p, err := s.find(game, profileID)
		if err != nil {
			return err
		}
		key, ok := winTarget(p, w)
		if !ok {
			continue
		}
		if _, err := s.d.Profiles.SetWinner(game, profileID, key, w.Winner, w.Loser, true); err != nil {
			log.Printf("share: %s/%s: could not make %s win over %s: %v", game, profileID, w.Winner, w.Loser, err)
		}
	}
	return nil
}

// winTarget is the key of the profile's entry for the win's download, when the profile has both packs and has not
// already decided the pair.
func winTarget(p profile.Profile, w share.Win) (string, bool) {
	hasLoser, reversed := false, false
	key := ""
	for _, e := range p.Entries {
		for _, m := range e.Mods {
			if mod.Equal(m.ID, w.Loser) {
				hasLoser = true
				reversed = reversed || slices.ContainsFunc(m.LoadAfter, func(id mod.ID) bool { return mod.Equal(id, w.Winner) })
			}
		}
		if w.Ref.MatchesEntry(e) {
			if i := slices.IndexFunc(e.Mods, func(m profile.Component) bool { return mod.Equal(m.ID, w.Winner) }); i >= 0 {
				if slices.ContainsFunc(e.Mods[i].LoadAfter, func(id mod.ID) bool { return mod.Equal(id, w.Loser) }) {
					return "", false
				}
				key = e.Key
			}
		}
	}
	return key, key != "" && hasLoser && !reversed
}

// withDismissed fills in the profile's dismissed problems when the share carries problem choices.
func (s *Service) withDismissed(game, profileID string, inc share.Include) share.Include {
	if inc.ProblemChoices && s.d.Dismissals != nil {
		inc.Dismissed = s.d.Dismissals.DismissedTokens(game, profileID)
	}
	return inc
}
