package backup

import (
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

// Target is where a game's save backups go and how many of each kind are kept.
type Target struct {
	// Dir receives new zips; Reads are the folders listed, Dir first.
	Dir   string
	Reads []string
	Keep  int
}

// TargetFor resolves (dataDir is the data folder, whose backups folder is the default location) the game's backupLocation and saveBackupsKept, honouring a profile's overrides. Every kind of
// save backup (update, restore, manual, launch) uses it, so they share one folder and one count.
func TargetFor(dataDir string, set settings.Settings, game string, overrides map[string]string) (Target, error) {
	dir, reads, err := Locations(dataDir, settings.ResolveAt(set, "backupLocation", settings.Scope{Game: game}, overrides), game)
	if err != nil {
		return Target{}, err
	}
	keep, err := strconv.Atoi(settings.ResolveAt(set, "saveBackupsKept", settings.Scope{Game: game}, overrides))
	if err != nil {
		keep = set.GamePrefs(game).SaveBackupsKept
	}
	return Target{Dir: dir, Reads: reads, Keep: keep}, nil
}
