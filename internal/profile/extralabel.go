package profile

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// ExtraFileLabel matches the GUI menu labels in frontend/src/mods/menu.ts.
func ExtraFileLabel(e Entry, extraKey string, files []nexus.File) string {
	_, fileID, nexusKey := store.NexusFile(extraKey)
	if nexusKey {
		for _, f := range files {
			if f.FileID == fileID {
				title := f.FileName
				if title == "" {
					title = f.Name
				}
				if f.Version != "" {
					return fmt.Sprintf("%s (%s)", title, f.Version)
				}
				return title
			}
		}
	}
	prefix := strings.ReplaceAll(extraKey, "\\", "/") + "/"
	var names []string
	for _, m := range e.Mods {
		folder := strings.ReplaceAll(m.Folder, "\\", "/")
		if folder != extraKey && !strings.HasPrefix(folder, prefix) {
			continue
		}
		if m.Version != "" {
			names = append(names, fmt.Sprintf("%s (%s)", m.Name, m.Version))
		} else {
			names = append(names, m.Name)
		}
	}
	if len(names) > 0 {
		return strings.Join(names, ", ")
	}
	return extraKey
}
