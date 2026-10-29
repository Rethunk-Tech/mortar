package meta

import "github.com/Rethunk-Tech/go-semverish"

// Newer reports whether a is a newer version than b; either failing to parse means not newer.
func Newer(a, b string) bool {
	c, ok := semverish.Compare(a, b)
	return ok && c > 0
}

// NewerFile reports whether a download file (version a, file id aID) is newer than another of the same mod: by
// version where both parse and differ, else the later upload, which has the higher file id (an author re-uploading a
// fix under the same version).
func NewerFile(a string, aID int64, b string, bID int64) bool {
	if c, ok := semverish.Compare(a, b); ok && c != 0 {
		return c > 0
	}
	return aID > bID
}
