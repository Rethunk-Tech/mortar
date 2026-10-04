package settings

import (
	"maps"
	"slices"
)

// RemoveDismissed drops tokens from bucket, so what they dismissed is offered again.
func (s *Store) RemoveDismissed(bucket string, tokens ...string) error {
	_, err := s.Update(func(v *Settings) {
		next := maps.Clone(v.Dismissed)
		next[bucket] = slices.DeleteFunc(slices.Clone(v.Dismissed[bucket]), func(t string) bool {
			return slices.Contains(tokens, t)
		})
		v.Dismissed = next
	})
	return err
}
