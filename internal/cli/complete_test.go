package cli

import (
	"slices"
	"testing"
)

func TestSubverbsIncludeDispatchedSubs(t *testing.T) {
	for verb, m := range map[string]map[string]func(*cmd) error{
		"profile": profileDispatchSubs,
		"mods":    modsDispatchSubs,
	} {
		for sub := range m {
			if !slices.Contains(subverbs[verb], sub) {
				t.Errorf("subverbs[%q] missing dispatched %q", verb, sub)
			}
		}
	}
}
