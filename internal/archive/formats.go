package archive

import (
	"path/filepath"
	"strings"

	ga "github.com/Rethunk-Tech/go-archive"
)

// HasExtension reports whether name ends in one of the archive endings Mortar offers, ignoring case.
func HasExtension(name string) bool { return ga.HasExtension(name) }

// StripExtension is name without its archive ending, or name itself when it has none.
func StripExtension(name string) string { return ga.StripExtension(name) }

// PickerPattern is the archive endings as a file dialog's semicolon-separated glob list.
func PickerPattern() string {
	exts := ga.Extensions()
	globs := make([]string, len(exts))
	for i, ext := range exts {
		globs[i] = "*" + ext
	}
	return CaseGlobs(strings.Join(globs, ";"))
}

// CaseGlobs adds an upper-case variant of each glob in a semicolon-separated list: GTK file filters match case
// sensitively, so "*.png" alone hides "A.PNG".
func CaseGlobs(pattern string) string {
	var out []string
	for g := range strings.SplitSeq(pattern, ";") {
		out = append(out, g)
		if up := strings.ToUpper(g); up != g {
			out = append(out, up)
		}
	}
	return strings.Join(out, ";")
}

// nameSep splits a saved download's file name into the queue id before it and the file's original name after it.
const nameSep = "~~"

// SavedName is the file name a download of fileName is kept under for id. A compressed single file keeps its
// original name after the id, because extraction names the one file it holds from that; every other download keeps
// its whole archive ending (".tar.gz", not ".gz"), which picks the extractor.
func SavedName(id, fileName string) string {
	if !ga.IsStream(fileName) {
		return id + fileName[len(StripExtension(fileName)):]
	}
	name := strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`/\<>:"|?*`, r) {
			return '_'
		}
		return r
	}, filepath.Base(fileName))
	if len(name) > maxSavedName {
		ext := filepath.Ext(name)
		name = name[:maxSavedName-len(ext)] + ext
	}
	return id + nameSep + name
}

// IsSaved reports whether name is one SavedName could have made: an archive ending, or the id separator.
func IsSaved(name string) bool { return HasExtension(name) || strings.Contains(name, nameSep) }

// originalName is the name a saved download had before SavedName prefixed its queue id. Extraction names a bare
// stream's one file from it.
func originalName(archivePath string) string {
	base := filepath.Base(archivePath)
	if _, original, ok := strings.Cut(base, nameSep); ok {
		return original
	}
	return base
}

// SavedID is the queue id a name from SavedName (or a resume sidecar's stem) belongs to.
func SavedID(name string) string {
	if id, _, ok := strings.Cut(name, nameSep); ok {
		return id
	}
	if stem := StripExtension(name); stem != name {
		return stem
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

const maxSavedName = 150
