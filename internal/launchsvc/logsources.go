package launchsvc

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// hiddenLogSourcesFile lists, one "source<TAB>object" per line, the call sites whose log lines the bridge drops in the
// game; an empty object drops the call site's lines about every object. Engine messages are hidden in the Console only.
const hiddenLogSourcesFile = startupDir + "/hidden-log-sources"

func (s *Service) writeHiddenLogSources(gameID, dir string) error {
	var b strings.Builder
	if s.settings != nil {
		for _, h := range s.settings.Get().GamePrefs(gameID).HiddenLogSources {
			if strings.Contains(h.Source, "::") {
				b.WriteString(h.Source + "\t" + h.Object + "\n")
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
