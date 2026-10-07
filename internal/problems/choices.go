package problems

import (
	"maps"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// DismissedTokens are the problems the player dismissed for the profile, and the setting hints they settled.
func (s *Service) DismissedTokens(gameID, id string) []string {
	return slices.Clone(s.settings.Get().Dismissed[dismissBucket(gameID, id)])
}

// AdoptDismissed adds tokens to the profile's dismissed problems. Ones it already holds stay as they are, and none
// is removed: the profile's own choices outlast what a share says.
func (s *Service) AdoptDismissed(gameID, id string, tokens []string) error {
	bucket := dismissBucket(gameID, id)
	_, err := s.settings.Update(func(v *settings.Settings) {
		have := v.Dismissed[bucket]
		add := slices.DeleteFunc(slices.Clone(tokens), func(t string) bool { return slices.Contains(have, t) })
		if len(add) == 0 {
			return
		}
		next := maps.Clone(v.Dismissed)
		next[bucket] = append(slices.Clone(have), slices.Compact(slices.Sorted(slices.Values(add)))...)
		v.Dismissed = next
	})
	return err
}
