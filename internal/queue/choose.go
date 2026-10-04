package queue

import (
	"regexp"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/nexus"
)

const (
	categoryMain     = "MAIN"
	categoryOptional = "OPTIONAL"
)

var fileVersionSuffix = regexp.MustCompile(`(?i)(?:[\s._-]+v?\d+(?:[._-]\d+)+|[\s._-]+v?\d+)$`)

func fileStem(name string) string {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)
	for _, ext := range []string{".zip", ".rar", ".7z"} {
		if strings.HasSuffix(lower, ext) {
			name = strings.TrimSpace(name[:len(name)-len(ext)])
			break
		}
	}
	for {
		stem := strings.TrimSpace(fileVersionSuffix.ReplaceAllString(name, ""))
		if stem == name {
			break
		}
		name = stem
	}
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// sameFileGroup reports whether two files are versions of the same download: the author's file_updates chain links
// them, they share Nexus's display name, or their archive names match once versions are stripped. Archive names
// alone are not enough: Nexus now appends an upload time and a random token to them.
func sameFileGroup(a, b nexus.File) bool {
	if a.ReplacedBy == b.FileID || b.ReplacedBy == a.FileID {
		return true
	}
	if a.Name != "" && b.Name != "" {
		return strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name))
	}
	aStem, bStem := fileStem(a.FileName), fileStem(b.FileName)
	return aStem == "" || bStem == "" || aStem == bStem
}

func betterFile(candidate, current nexus.File) bool {
	candidateOld := strings.EqualFold(candidate.Category, "OLD_VERSION")
	currentOld := strings.EqualFold(current.Category, "OLD_VERSION")
	if candidateOld != currentOld {
		return currentOld
	}
	return candidate.FileID > current.FileID
}

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
	currentFile := fileByID(files, current)
	// The author's own chain from the installed file is the surest answer for an update.
	if currentFile.FileID != 0 {
		if next := newestUpdate(files, currentFile); next.FileID != currentFile.FileID &&
			(version == "" || sameVersion(next, version)) {
			return next, true
		}
	}
	for _, f := range files {
		if f.FileID == current && strings.EqualFold(f.Category, categoryOptional) {
			categories = []string{categoryOptional, categoryMain}
		}
	}
	if version != "" {
		for _, category := range categories {
			var best nexus.File
			for _, f := range files {
				if strings.EqualFold(f.Category, category) && sameVersion(f, version) &&
					(currentFile.FileID == 0 || sameFileGroup(f, currentFile)) && f.FileID > best.FileID {
					best = f
				}
			}
			if best.FileID != 0 {
				return best, true
			}
		}
	}
	for _, f := range files {
		if f.IsPrimary && (!strings.EqualFold(f.Category, categoryOptional) || categories[0] == categoryOptional) &&
			(currentFile.FileID == 0 || sameFileGroup(f, currentFile)) {
			return f, true
		}
	}
	return nexus.File{}, false
}

func fileByID(files []nexus.File, id int) nexus.File {
	for _, f := range files {
		if f.FileID == id {
			return f
		}
	}
	return nexus.File{}
}

// newestUpdate follows the author's file_updates chain from file to the newest file still listed.
func newestUpdate(files []nexus.File, file nexus.File) nexus.File {
	seen := map[int]bool{}
	for file.ReplacedBy != 0 && !seen[file.FileID] {
		seen[file.FileID] = true
		next := fileByID(files, file.ReplacedBy)
		if next.FileID == 0 {
			break
		}
		if !sameFileGroup(file, next) {
			break
		}
		file = next
	}
	if fileStem(file.FileName) != "" {
		for _, candidate := range files {
			if sameFileGroup(candidate, file) && betterFile(candidate, file) {
				file = candidate
			}
		}
	}
	return file
}
