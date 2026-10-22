//go:build linux

package datasvc

import (
	"os"
	"reflect"
	"runtime"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"golang.org/x/sys/unix"
)

const (
	fiemapExtentLast       = 0x00000001
	fiemapExtentUnknown    = 0x00000002
	fiemapExtentDelalloc   = 0x00000004
	fiemapExtentNotAligned = 0x00000100
	fiemapExtentDataInline = 0x00000200
	fiemapExtentBad        = fiemapExtentUnknown | fiemapExtentDelalloc | fiemapExtentNotAligned | fiemapExtentDataInline
	fiemapExtentBatch      = 64
	// fsIocFiemap is FS_IOC_FIEMAP (_IOWR('f', 11, struct fiemap)).
	fsIocFiemap = 0xc020660b
)

type physKey struct {
	dev, physical uint64
}

type shareBlocks struct {
	phys map[physKey]struct{}
}

func newShareBlocks() shareBlocks {
	return shareBlocks{phys: map[physKey]struct{}{}}
}

type fiemapExtent struct {
	Logical    uint64
	Physical   uint64
	Length     uint64
	Reserved64 [2]uint64
	Flags      uint32
	Reserved   [3]uint32
}

type fiemapQuery struct {
	Start         uint64
	Length        uint64
	Flags         uint32
	MappedExtents uint32
	ExtentCount   uint32
	Reserved      uint32
	Extents       [fiemapExtentBatch]fiemapExtent
}

// extentRun is one allocated range of a file, as FIEMAP reports it.
type extentRun struct {
	physical, length uint64
}

// fileProbe is a file's extent map, read apart from the sharing accounting so files can be read concurrently.
type fileProbe struct {
	runs []extentRun
	// ok is false when the filesystem cannot say (no FIEMAP, or extents that are not plain allocations).
	ok bool
	// endedEmpty is set when the map ended on an empty answer rather than the last extent.
	endedEmpty bool
}

func probeFile(path string) fileProbe {
	f, err := fsx.Open(path)
	if err != nil {
		return fileProbe{}
	}
	defer func() { _ = f.Close() }()
	return readExtents(f)
}

func fileExclusive(info os.FileInfo, p fileProbe, s *shareAcc) (int64, bool) {
	dev, ino, alloc, ok := fileAlloc(info)
	if !ok {
		return info.Size(), false
	}
	if !p.ok {
		return inodeExclusive(dev, ino, alloc, s)
	}
	n, ok := foldExtents(p, dev, info.Size(), s)
	if !ok {
		return inodeExclusive(dev, ino, alloc, s)
	}
	return n, true
}

func inodeExclusive(dev, ino uint64, alloc int64, s *shareAcc) (int64, bool) {
	k := inodeKey{dev, ino}
	if _, dup := s.seen[k]; dup {
		return 0, true
	}
	s.seen[k] = struct{}{}
	return alloc, true
}

func readExtents(f *os.File) fileProbe {
	var (
		start uint64
		p     fileProbe
	)
	for {
		var q fiemapQuery
		q.Start = start
		q.Length = ^uint64(0)
		q.ExtentCount = fiemapExtentBatch
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), fsIocFiemap, reflect.ValueOf(&q).Pointer())
		runtime.KeepAlive(&q)
		if errno != 0 {
			return fileProbe{}
		}
		if q.MappedExtents == 0 {
			p.ok, p.endedEmpty = true, true
			return p
		}
		var last bool
		for i := uint32(0); i < q.MappedExtents; i++ {
			e := q.Extents[i]
			if e.Flags&fiemapExtentBad != 0 {
				return fileProbe{}
			}
			p.runs = append(p.runs, extentRun{e.Physical, e.Length})
			if e.Flags&fiemapExtentLast != 0 {
				last = true
			}
			start = e.Logical + e.Length
		}
		if last || q.MappedExtents < fiemapExtentBatch {
			p.ok = true
			return p
		}
	}
}

// foldExtents is the bytes of p's extents no earlier file claimed.
func foldExtents(p fileProbe, dev uint64, size int64, s *shareAcc) (int64, bool) {
	var cost int64
	for _, r := range p.runs {
		k := physKey{dev, r.physical}
		if _, dup := s.blocks.phys[k]; dup {
			continue
		}
		s.blocks.phys[k] = struct{}{}
		n, err := strconv.ParseInt(strconv.FormatUint(r.length, 10), 10, 64)
		if err != nil {
			return 0, false
		}
		cost += n
	}
	if p.endedEmpty {
		return cost, size == 0 || cost > 0
	}
	return cost, true
}
