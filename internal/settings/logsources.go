package settings

import "slices"

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

func (h HiddenLogSource) same(o HiddenLogSource) bool {
	return h.Source == o.Source && h.Text == o.Text && h.Object == o.Object
}

// HideLogSource hides a source of the game's log lines from the Console, and from the game's log where its bridge can.
func (s *Service) HideLogSource(game string, src HiddenLogSource) error {
	return s.set(func(v *Settings) {
		gp := v.GamePrefs(game)
		if src.Source != "" && !slices.ContainsFunc(gp.HiddenLogSources, src.same) {
			gp.HiddenLogSources = append(slices.Clone(gp.HiddenLogSources), src)
		}
		putGame(v, game, gp)
	})
}

// ShowLogSource shows a hidden source's lines again.
func (s *Service) ShowLogSource(game, source, text, object string) error {
	return s.set(func(v *Settings) {
		gp := v.GamePrefs(game)
		gp.HiddenLogSources = slices.DeleteFunc(slices.Clone(gp.HiddenLogSources), HiddenLogSource{Source: source, Text: text, Object: object}.same)
		putGame(v, game, gp)
	})
}
