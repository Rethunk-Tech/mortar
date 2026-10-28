package launchsvc

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// hiddenLogSourcesFile lists, one "method<TAB>text<TAB>object" per line, the log lines the bridge drops in the game:
// those the method logs whose first line masks to text, about that object or, when it is empty, any. Engine messages
// are hidden in the Console only, and a mod's messages by its log column in the parser (settings.HideModMessages).
const hiddenLogSourcesFile = startupDir + "/hidden-log-sources"

func (s *Service) writeHiddenLogSources(ctx context.Context, gameID, dir string) error {
	var b strings.Builder
	if s.settings != nil {
		hidden := settings.HiddenLogSources(s.settings.Get().GamePrefs(gameID), settings.LogSuppressionDefaults(ctx, gameID))
		settings.HideModMessages(gameID, hidden)
		for _, h := range hidden {
			if strings.Contains(h.Source, "::") {
				b.WriteString(h.Source + "\t" + h.Text + "\t" + h.Object + "\n")
			}
		}
	}
	path := filepath.Join(dir, filepath.FromSlash(hiddenLogSourcesFile))
	return datadir.WriteOrRemove(path, []byte(b.String()))
}

// PrimeHiddenMessages hands the log parser every set-up game's hidden mod messages, before any log is read.
func (s *Service) PrimeHiddenMessages(ctx context.Context) {
	if s.settings == nil {
		return
	}
	v := s.settings.Get()
	for id := range v.Games {
		settings.HideModMessages(id, settings.HiddenLogSources(v.GamePrefs(id), settings.LogSuppressionDefaults(ctx, id)))
	}
}
