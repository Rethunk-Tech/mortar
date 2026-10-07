package archive

import (
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
