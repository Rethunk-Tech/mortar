package datasvc

import (
	"os"
)

type inodeKey struct {
	dev, ino uint64
}

type shareAcc struct {
	logical, allocated int64
	seen               map[inodeKey]struct{}
	blocks             shareBlocks
	ok                 bool
}

func newShareAcc() *shareAcc {
	return &shareAcc{seen: map[inodeKey]struct{}{}, blocks: newShareBlocks()}
}

func (s *shareAcc) addFollowed(path string, info os.FileInfo) {
	s.addSize(path, info, false)
}

func (s *shareAcc) addSize(path string, info os.FileInfo, countAlloc bool) {
	dev, ino, alloc, ok := fileAlloc(path, info)
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
