package queue

import (
	"strings"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
)

const (
	categoryMain     = "MAIN"
	categoryOptional = "OPTIONAL"
)

func sameVersion(f nexus.File, want string) bool {
	for _, v := range []string{f.Version, f.ModVersion} {
		if v == want {
			return true
		}
		if c, ok := meta.CompareVersions(v, want); ok && c == 0 {
			return true
		}
	}
	return false
}

// ChooseFile picks the file of a mod to download: the MAIN file at version, else the primary file. An OPTIONAL
// file is only ever chosen when current, the file the profile has now, is one, in which case an OPTIONAL file at
// version comes first. version may be empty, which leaves only the primary file.
func ChooseFile(files []nexus.File, version string, current int) (nexus.File, bool) {
	categories := []string{categoryMain}
	for _, f := range files {
		if f.FileID == current && strings.EqualFold(f.Category, categoryOptional) {
			categories = []string{categoryOptional, categoryMain}
		}
	}
	if version != "" {
		for _, category := range categories {
			var best nexus.File
			for _, f := range files {
				if strings.EqualFold(f.Category, category) && sameVersion(f, version) && f.FileID > best.FileID {
					best = f
				}
			}
			if best.FileID != 0 {
				return best, true
			}
		}
	}
	for _, f := range files {
		if f.IsPrimary && (!strings.EqualFold(f.Category, categoryOptional) || categories[0] == categoryOptional) {
			return f, true
		}
	}
	return nexus.File{}, false
}
