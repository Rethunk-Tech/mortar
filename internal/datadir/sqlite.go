package datadir

import (
	"net/url"
	"path/filepath"
)

// ReadOnlySQLiteURI is SQLite's read-only, immutable URI for a database path, so reading never disturbs the program
// that owns the file (what it has not yet checkpointed from its write-ahead log is not seen). A Windows path ("C:/...")
// needs the leading slash too, or SQLite takes the drive for the URI's authority and refuses it.
func ReadOnlySQLiteURI(db string) string {
	p := filepath.ToSlash(db)
	if len(p) > 1 && p[1] == ':' {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p, RawQuery: "mode=ro&immutable=1"}).String()
}
