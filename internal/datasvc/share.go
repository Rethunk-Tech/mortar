package datasvc

import (
	"os"
	"path/filepath"
	"strings"
)

type inodeKey struct {
	dev, ino uint64
}

type shareAcc struct {
	logical, allocated int64
	seen               map[inodeKey]struct{}
	ok                 bool
}

func newShareAcc() *shareAcc {
	return &shareAcc{seen: map[inodeKey]struct{}{}}
}

func (s *shareAcc) add(info os.FileInfo) {
	s.addSize(info, true)
}

func (s *shareAcc) addFollowed(info os.FileInfo) {
	s.addSize(info, false)
}

func (s *shareAcc) addSize(info os.FileInfo, countAlloc bool) {
	dev, ino, alloc, ok := fileAlloc(info)
	if !ok {
		return
	}
	s.ok = true
	s.logical += info.Size()
	if !countAlloc {
		return
	}
	k := inodeKey{dev, ino}
	if _, dup := s.seen[k]; dup {
		return
	}
	s.seen[k] = struct{}{}
	s.allocated += alloc
}

func (s *shareAcc) saved() (n int64, known bool) {
	if !s.ok {
		return 0, false
	}
	return max(s.logical-s.allocated, 0), true
}

func under(root, p string) bool {
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	if root == p {
		return true
	}
	return strings.HasPrefix(p, root+string(os.PathSeparator))
}
