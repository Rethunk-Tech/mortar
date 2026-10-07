package archive

import (
	"path/filepath"
	"slices"
	"strings"
)

// Extensions are the file name endings Mortar offers as archives, longest first so a compound ending wins over its
// last part. Detection never trusts them: the magic bytes decide the format.
var Extensions = []string{
	".tar.gz", ".tar.xz", ".tar.zst", ".tar.bz2", ".tar.lzma",
	".tgz", ".txz", ".tzst", ".tbz2", ".tbz",
	".zip", ".rar", ".7z", ".tar", ".gz", ".xz", ".zst", ".lzma", ".bz2",
}

// HasExtension reports whether name ends in one of Extensions, ignoring case.
func HasExtension(name string) bool {
	_, ok := cutExtension(name)
	return ok
}

// StripExtension is name without its archive ending, or name itself when it has none.
func StripExtension(name string) string {
	if stem, ok := cutExtension(name); ok {
		return stem
	}
	return name
}

func cutExtension(name string) (string, bool) {
	lower := strings.ToLower(name)
	for _, ext := range Extensions {
		if strings.HasSuffix(lower, ext) {
			return name[:len(name)-len(ext)], true
		}
	}
	return name, false
}

// PickerPattern is Extensions as a file dialog's semicolon-separated glob list.
func PickerPattern() string {
	globs := make([]string, len(Extensions))
	for i, ext := range Extensions {
		globs[i] = "*" + ext
	}
	return strings.Join(globs, ";")
}

// nameSep splits a saved download's file name into the queue id before it and the file's original name after it.
const nameSep = "~~"

// bareSuffixes are the endings of a compressed single file (a tar is not one: its entries carry their own names).
var bareSuffixes = []string{".gz", ".xz", ".zst", ".lzma", ".bz2"}

func isBare(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range Extensions {
		if strings.HasSuffix(lower, ext) {
			return slices.Contains(bareSuffixes, ext)
		}
	}
	return false
}

// SavedName is the file name a download of fileName is kept under for id. A compressed single file keeps its
// original name after the id, because extraction names the one file it holds from that; every other download keeps
// just its extension.
func SavedName(id, fileName string) string {
	if !isBare(fileName) {
		return id + filepath.Ext(fileName)
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

// SavedID is the queue id a name from SavedName (or a resume sidecar's stem) belongs to.
func SavedID(name string) string {
	if id, _, ok := strings.Cut(name, nameSep); ok {
		return id
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

const maxSavedName = 150
