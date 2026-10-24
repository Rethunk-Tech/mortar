package settings

import (
	"context"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// HiddenLogSource is a source of game log lines hidden from the Console: the method that logged them (Type::Method,
// from the bridge) with Text, their first line with its numbers masked as "#" (one method logs many different lines),
// or for a message the engine logs itself, "unity:" and that masked line. Object narrows it to the lines about one
// object (its hierarchy path), empty is every object. Label is a line it hid, for the list that shows it again.
type HiddenLogSource struct {
	Source string `json:"source"`
	Text   string `json:"text,omitempty"`
	Object string `json:"object,omitempty"`
	Label  string `json:"label"`
}

// covers reports whether hiding h already hides every line o would: the same source, any text or o's, any object or
// o's.
func (h HiddenLogSource) covers(o HiddenLogSource) bool {
	return h.Source == o.Source && (h.Text == "" || h.Text == o.Text) && (h.Object == "" || h.Object == o.Object)
}

func (h HiddenLogSource) same(o HiddenLogSource) bool {
	return h.Source == o.Source && h.Text == o.Text && h.Object == o.Object
}

// HideLogSource hides a source of the game's log lines from the Console, and from the game's log where its bridge can.
// A source already hidden by a broader one adds nothing; a broader one replaces the narrower ones it covers.
func (s *Service) HideLogSource(game string, src HiddenLogSource) error {
	return s.set(func(v *Settings) {
		gp := v.GamePrefs(game)
		covered := slices.ContainsFunc(gp.HiddenLogSources, func(h HiddenLogSource) bool { return h.covers(src) })
		if src.Source != "" && !covered {
			gp.HiddenLogSources = append(slices.DeleteFunc(slices.Clone(gp.HiddenLogSources), src.covers), src)
		}
		gp.ShownLogSources = slices.DeleteFunc(slices.Clone(gp.ShownLogSources), src.covers)
		putGame(v, game, gp)
	})
}

// ShowLogSource shows a hidden source's lines again: one the player hid leaves their list, and one only the game's
// default suppressions hide joins the list of defaults shown again.
func (s *Service) ShowLogSource(game, source, text, object string) error {
	show := HiddenLogSource{Source: source, Text: text, Object: object}
	return s.set(func(v *Settings) {
		gp := v.GamePrefs(game)
		n := len(gp.HiddenLogSources)
		gp.HiddenLogSources = slices.DeleteFunc(slices.Clone(gp.HiddenLogSources), show.same)
		if len(gp.HiddenLogSources) == n && !slices.ContainsFunc(gp.ShownLogSources, show.same) {
			gp.ShownLogSources = append(slices.Clone(gp.ShownLogSources), show)
		}
		putGame(v, game, gp)
	})
}

// LogSuppressionDefaults are the sources of game log lines the game's published list hides unless the player shows
// them again.
func (s *Service) LogSuppressionDefaults(ctx context.Context, game string) []HiddenLogSource {
	return LogSuppressionDefaults(ctx, game)
}

// LogSuppressionDefaults are the sources of game log lines the game's published list hides by default.
func LogSuppressionDefaults(ctx context.Context, game string) []HiddenLogSource {
	list := components.LogSuppressionsFor(ctx, game)
	out := make([]HiddenLogSource, 0, len(list.Entries))
	for _, e := range list.Entries {
		out = append(out, HiddenLogSource(e))
	}
	return out
}

// HiddenLogSources are every source of the game's log lines hidden, each once: the defaults the player has not shown
// again, then the player's own.
func HiddenLogSources(gp GameSettings, defaults []HiddenLogSource) []HiddenLogSource {
	out := slices.DeleteFunc(slices.Clone(defaults), func(d HiddenLogSource) bool {
		return slices.ContainsFunc(gp.ShownLogSources, d.same) || slices.ContainsFunc(gp.HiddenLogSources, d.same)
	})
	return append(out, gp.HiddenLogSources...)
}
