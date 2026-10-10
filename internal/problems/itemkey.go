package problems

import (
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/store"
)

// itemOf is the store item an entry key names: a per-file entry of a folder game is `<store item>#<path>`, and every
// store lookup, update check and duplicate count is about the item.
func itemOf(key string) string {
	item, _, _ := strings.Cut(key, "#")
	return item
}

// nexusFile is store.NexusFile of the key's store item.
func nexusFile(key string) (modID, fileID int, ok bool) { return store.NexusFile(itemOf(key)) }

// oncePerItem keeps the first update of each store item and version: the files an archive split into share one
// download, so offering the update once per file would count it several times.
func oncePerItem(updates []Update) []Update {
	type seen struct {
		item, version string
		unofficial    bool
	}
	var done []seen
	return slices.DeleteFunc(updates, func(u Update) bool {
		if !strings.Contains(u.Key, "#") {
			return false
		}
		s := seen{itemOf(u.Key), u.Version, u.Unofficial}
		if slices.Contains(done, s) {
			return true
		}
		done = append(done, s)
		return false
	})
}
