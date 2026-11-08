package saves

import (
	"context"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-vintagestory/savedb"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// readWorld fills a Vintage Story world's name, play style, game version, play time and the mods it records from the one
// gamedata row of its .vcdbs, a SQLite database holding a protobuf SaveGame.
func readWorld(ctx context.Context, file string, info *Info) error {
	w, err := savedb.Read(ctx, file)
	if err != nil {
		return err
	}
	info.Name, info.PlayStyle, info.GameVersion, info.WorldID = w.Name, w.PlayStyle, w.GameVersion, w.WorldID
	info.MillisecondsPlayed = w.Played.Milliseconds()
	info.Used = info.Used[:0]
	for _, d := range w.Domains {
		info.Used = append(info.Used, mod.NewID(mod.FormatModInfo, d))
	}
	return nil
}

// worldIDPattern is the shape of a world's SavegameIdentifier (a GUID), which names files outside the saves folder.
var worldIDPattern = regexp.MustCompile(`^[0-9A-Fa-f-]{1,64}$`)

// LinkedOf lists the files outside the saves folder that belong to save name, as slash paths under Dir's parent:
// each Linked pattern with {uuid} filled from the world's own identifier (a Vintage Story world's minimap,
// Maps/<uuid>.db). Only those that exist are listed.
func (l Layout) LinkedOf(ctx context.Context, name string) []string {
	if len(l.Linked) == 0 || !l.IsSave(name) || !strings.EqualFold(path.Ext(name), ".vcdbs") {
		return nil
	}
	var info Info
	if readWorld(ctx, filepath.Join(l.Dir, filepath.FromSlash(name)), &info) != nil || !worldIDPattern.MatchString(info.WorldID) {
		return nil
	}
	var out []string
	for _, p := range l.Linked {
		rel := strings.ReplaceAll(p, "{uuid}", info.WorldID)
		if safefs.IsFile(filepath.Join(filepath.Dir(l.Dir), filepath.FromSlash(rel))) {
			out = append(out, rel)
		}
	}
	return out
}
