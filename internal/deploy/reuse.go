package deploy

import (
	"crypto/sha256"
	"os"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Reuse is how a Play decides that a file the previous Play placed still stands and can stay where it is.
type Reuse int

const (
	// ReuseNone places every file again and takes them all back at the end of the Play. The zero value.
	ReuseNone Reuse = iota
	// ReuseStat keeps a file whose source and copy still have the size and modification time the record took.
	ReuseStat
)

// previous reads the journal of a Play that left its files in place: the record by folded destination, for the files
// the next Play may keep.
func previous(dir string) (Manifest, map[string]int, bool, error) {
	var m Manifest
	found, err := datadir.ReadJournal(journalPath(dir), nil, &m)
	if err != nil || !found {
		return Manifest{}, nil, false, err
	}
	if err := replay(&m); err != nil {
		return Manifest{}, nil, false, err
	}
	byDst := make(map[string]int, len(m.Ops))
	for i, o := range m.Ops {
		byDst[safefs.FoldCase(o.Dst)] = i
	}
	return m, byDst, true, nil
}

// stands reports whether the file the previous Play put at rec.Dst is the one this Play would put there: the copy is
// untouched, and the source it came from has not changed since.
func (r Reuse) stands(rec Placed) bool {
	if r == ReuseNone || !rec.Done {
		return false
	}
	if !r.sameCopy(rec) {
		return false
	}
	src, err := os.Stat(rec.Src)
	return err == nil && src.Size() == rec.SrcSize && src.ModTime().UnixNano() == rec.SrcMTime
}

// sameCopy reports whether what is at rec.Dst is still the file the record describes: a game or a mod that rewrites a
// placed file moves its size or its time.
func (r Reuse) sameCopy(rec Placed) bool {
	dst, err := os.Lstat(rec.Dst)
	if err != nil || !dst.Mode().IsRegular() || dst.Size() != rec.Size || dst.ModTime().UnixNano() != rec.MTime {
		return false
	}
	return true
}

// sourceHash is the source's hash, taken from the record when its size and time still match, so a second Play reads
// only the files that changed.
func sourceHash(src string, rec Placed, have bool) (hash string, size, mtime int64, err error) {
	st, err := os.Stat(src)
	if err != nil {
		return "", 0, 0, err
	}
	if have && st.Size() == rec.SrcSize && st.ModTime().UnixNano() == rec.SrcMTime {
		return rec.Hash, rec.SrcSize, rec.SrcMTime, nil
	}
	h, err := safefs.HashFile(src, sha256.New())
	if err != nil {
		return "", 0, 0, err
	}
	return h, st.Size(), st.ModTime().UnixNano(), nil
}
