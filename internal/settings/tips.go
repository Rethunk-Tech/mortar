package settings

import (
	"fmt"
	"slices"
)

var knownTips = []string{"mods", "saves", "console", "share", "tour"}

func knownTip(id string) bool {
	return slices.Contains(knownTips, id)
}

func sanitizeTips(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if !knownTip(id) || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func validateTips(s Settings) error {
	for _, id := range s.TipsSeen {
		if !knownTip(id) {
			return fmt.Errorf("unknown tip %q", id)
		}
	}
	return nil
}

func normalizeTips(s *Settings) {
	s.TipsSeen = sanitizeTips(s.TipsSeen)
}
