package control

import (
	"context"
	"strings"
	"testing"
)

// computedCLIMethods are the names the CLI builds as prefix+subcommand, so no map
// key in this package holds them as a single literal. Each list is the subcommand
// set of the switch that builds the name in internal/cli.
var computedCLIMethods = map[string][]string{
	"mods.": {
		"enable", "disable", "remove", "pin", "unpin", "tag", "untag", "category", "note", "skip-version", "split", "combine",
		"favourite", "unfavourite", "favourites",
	},
	"profile.": {"freeze", "unfreeze", "favourite", "unfavourite"},
	"queue.":   {"retry", "skip", "pause-item", "resume-item", "top", "up", "down", "pause", "resume", "clear"},
	"backups.": {"keep", "unkeep"},
}

func TestEveryMethodTheCLISendsIsDispatched(t *testing.T) {
	have := make(map[string]bool, len(globalMethods)+len(profileMethods))
	for m := range globalMethods {
		have[m] = true
	}
	for m := range profileMethods {
		have[m] = true
	}
	if len(have) < 70 {
		t.Fatalf("dispatch maps have only %d methods", len(have))
	}
	for prefix, subs := range computedCLIMethods {
		for _, sub := range subs {
			m := prefix + sub
			if !have[m] {
				t.Errorf("the CLI sends %q and no handler answers it", m)
			}
		}
	}
}

func TestGameNameUnknownReturnsID(t *testing.T) {
	name, ok := gameName("not-a-game")
	if ok {
		t.Fatal("unknown game reported as found")
	}
	if name != "not-a-game" {
		t.Fatalf("got %q, want the id", name)
	}
}

func TestUnknownMethodError(t *testing.T) {
	s := services(t)
	_, err := s.Handle(context.Background(), "not.a.method", Params{})
	if err == nil || !strings.HasPrefix(err.Error(), "unknown method") {
		t.Fatalf("got %v, want unknown method", err)
	}
}
