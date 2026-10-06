package nexus

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
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
func sameFileGroup(a, b File) bool {
	if a.ReplacedBy == b.FileID || b.ReplacedBy == a.FileID {
		return true
	}
	if a.Name != "" && b.Name != "" {
		return strings.EqualFold(strings.TrimSpace(a.Name), strings.TrimSpace(b.Name))
	}
	aStem, bStem := fileStem(a.FileName), fileStem(b.FileName)
	return aStem == "" || bStem == "" || aStem == bStem
}

func betterFile(candidate, current File) bool {
	candidateOld := strings.EqualFold(candidate.Category, "OLD_VERSION")
	currentOld := strings.EqualFold(current.Category, "OLD_VERSION")
	if candidateOld != currentOld {
		return currentOld
	}
	return candidate.FileID > current.FileID
}

// Stale reports a file Nexus keeps listed only as history.
func Stale(f File) bool {
	switch strings.ToUpper(f.Category) {
	case "OLD_VERSION", "ARCHIVED", "DELETED", "REMOVED":
		return true
	}
	return false
}

func sameVersion(f File, want string) bool {
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

// newer reports whether f is a later version than than, by version number where both parse, else by upload order.
func newer(f, than File) bool {
	if c, ok := meta.CompareVersions(f.Version, than.Version); ok {
		return c > 0
	}
	return f.FileID > than.FileID
}

// only is the one file match accepts, if exactly one does.
func only(files []File, match func(File) bool) (File, bool) {
	var hit File
	n := 0
	for _, f := range files {
		if match(f) {
			hit, n = f, n+1
		}
	}
	return hit, n == 1
}

// Supersedes picks the file an update of installed downloads. A mod page can carry several downloads (a main file
// and optional packs), so the newest file of the mod is not the answer. In order: the author's file_updates chain
// from installed; a newer file of the same download (display name or archive stem), the one at version first; the
// single current file of installed's kind at version; and, only for a MAIN entry on a page with exactly one current
// MAIN file, that file. kind is the category the entry was installed as, empty to use installed's own: an archived
// file no longer says what it was. ok is false when nothing supersedes installed.
func Supersedes(files []File, installed File, version, kind string) (File, bool) {
	if kind == "" {
		kind = installed.Category
	}
	if next := NewestUpdate(files, installed); next.FileID != installed.FileID {
		return next, true
	}
	var group File
	for _, f := range files {
		// A MAIN entry never becomes an optional pack, whatever the names say.
		if f.FileID == installed.FileID || Stale(f) || !newer(f, installed) || !sameFileGroup(f, installed) ||
			(strings.EqualFold(f.Category, categoryOptional) && strings.EqualFold(kind, categoryMain)) {
			continue
		}
		atF, atGroup := version != "" && sameVersion(f, version), version != "" && sameVersion(group, version)
		if group.FileID == 0 || (atF && !atGroup) || (atF == atGroup && f.FileID > group.FileID) {
			group = f
		}
	}
	if group.FileID != 0 {
		return group, true
	}
	if version != "" {
		if f, ok := only(files, func(f File) bool {
			return !Stale(f) && strings.EqualFold(f.Category, kind) && sameVersion(f, version)
		}); ok && f.FileID != installed.FileID {
			return f, true
		}
	}
	if strings.EqualFold(kind, categoryMain) {
		if f, ok := only(files, func(f File) bool { return strings.EqualFold(f.Category, categoryMain) }); ok &&
			f.FileID != installed.FileID && newer(f, installed) {
			return f, true
		}
	}
	return File{}, false
}

// ChooseFile picks the file of a mod to download. With current, the file the profile has now, listed, it is the file
// that supersedes current (Supersedes), or current itself when it is at version. Otherwise it is the MAIN file at version, else the primary file; an
// OPTIONAL file is never chosen for a profile that has none. version may be empty, which leaves only the primary
// file.
func ChooseFile(files []File, version string, current int) (File, bool) {
	if installed := FileByID(files, current); installed.FileID != 0 {
		if f, ok := Supersedes(files, installed, version, ""); ok {
			return f, true
		}
		// The profile already has the file at version: downloading it again is a reinstall, not a guess.
		return installed, version != "" && sameVersion(installed, version)
	}
	if version != "" {
		var best File
		for _, f := range files {
			if strings.EqualFold(f.Category, categoryMain) && sameVersion(f, version) && f.FileID > best.FileID {
				best = f
			}
		}
		if best.FileID != 0 {
			return best, true
		}
	}
	for _, f := range files {
		if f.IsPrimary && !strings.EqualFold(f.Category, categoryOptional) {
			return f, true
		}
	}
	return File{}, false
}

// FileByID is the listed file with id, or the zero File.
func FileByID(files []File, id int) File {
	for _, f := range files {
		if f.FileID == id {
			return f
		}
	}
	return File{}
}

// NewestUpdate follows the author's file_updates chain from file to the newest file still listed.
func NewestUpdate(files []File, file File) File {
	seen := map[int]bool{}
	for file.ReplacedBy != 0 && !seen[file.FileID] {
		seen[file.FileID] = true
		next := FileByID(files, file.ReplacedBy)
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
