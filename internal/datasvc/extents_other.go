//go:build !linux

package datasvc

import "os"

func newShareBlocks() shareBlocks {
	return shareBlocks{}
}

type shareBlocks struct{}

func fileExclusive(path string, info os.FileInfo, s *shareAcc) (int64, bool) {
	_ = path
	_ = s.blocks
	dev, ino, _, ok := fileAlloc(info)
	if !ok {
		return info.Size(), false
	}
	k := inodeKey{dev, ino}
	if _, dup := s.seen[k]; dup {
		return 0, true
	}
	s.seen[k] = struct{}{}
	return info.Size(), true
}
