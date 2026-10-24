package launchsvc

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// hiddenLogSourcesFile lists, one "method<TAB>text<TAB>object" per line, the log lines the bridge drops in the game:
// those the method logs whose first line masks to text, about that object or, when it is empty, any. Engine messages
// are hidden in the Console only.
const hiddenLogSourcesFile = startupDir + "/hidden-log-sources"

func (s *Service) writeHiddenLogSources(ctx context.Context, gameID, dir string) error {
	var b strings.Builder
	if s.settings != nil {
		gp := s.settings.Get().GamePrefs(gameID)
		for _, h := range settings.HiddenLogSources(gp, settings.LogSuppressionDefaults(ctx, gameID)) {
			if strings.Contains(h.Source, "::") {
				b.WriteString(h.Source + "\t" + h.Text + "\t" + h.Object + "\n")
			}
		}
	}
	path := filepath.Join(dir, filepath.FromSlash(hiddenLogSourcesFile))
	if b.Len() == 0 {
		if err := fsx.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteFile(path, []byte(b.String()), 0o600)
}
