package larian

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pierrec/lz4/v4"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// The LSPK v18 layout Baldur's Gate 3 packs mods in (Norbyte/lslib Package.cs): a signature, a header naming the
// file list, and the list itself LZ4-compressed as fixed-size entries.
const (
	pakSignature = "LSPK"
	pakVersion   = 18
	entrySize    = 272
	nameSize     = 256
	// maxEntries and maxMeta bound what a pak from the internet may make Mortar allocate.
	maxEntries = 1 << 16
	maxMeta    = 1 << 20
	// maxOffset is past any pak a mod ships, and keeps every offset an int64.
	maxOffset = 1 << 40
)

const (
	compressNone = 0
	compressZlib = 1
	compressLZ4  = 2
)

var errNotPak = errors.New("not an LSPK v18 package")

type pakHeader struct {
	Version        uint32
	FileListOffset uint64
	FileListSize   uint32
	Flags          uint8
	Priority       uint8
	MD5            [16]byte
	NumParts       uint16
}

type pakEntry struct {
	name             string
	offset           int64
	part, flags      uint8
	size, uncompress uint32
}

// readPakFile reads the file of the pak at path whose slash path satisfies match, or reports false when none does.
func readPakFile(path string, match func(name string) bool) ([]byte, string, bool, error) {
	f, err := fsx.Open(path)
	if err != nil {
		return nil, "", false, err
	}
	defer func() { _ = f.Close() }()
	entries, err := pakEntries(f)
	if err != nil {
		return nil, "", false, err
	}
	for _, e := range entries {
		if !match(e.name) {
			continue
		}
		if e.part != 0 {
			return nil, "", false, fmt.Errorf("%s is in another part of the package", e.name)
		}
		if e.size > maxMeta || e.uncompress > maxMeta {
			return nil, "", false, fmt.Errorf("%s is too large", e.name)
		}
		raw := make([]byte, e.size)
		if _, err := f.ReadAt(raw, e.offset); err != nil {
			return nil, "", false, err
		}
		b, err := decompress(raw, e.flags, int(e.uncompress))
		return b, e.name, err == nil, err
	}
	return nil, "", false, nil
}

func pakEntries(r io.ReaderAt) ([]pakEntry, error) {
	var sig [4]byte
	if _, err := r.ReadAt(sig[:], 0); err != nil || string(sig[:]) != pakSignature {
		return nil, errNotPak
	}
	var h pakHeader
	if err := binary.Read(io.NewSectionReader(r, 4, 64), binary.LittleEndian, &h); err != nil {
		return nil, err
	}
	if h.Version != pakVersion || h.FileListOffset > maxOffset {
		return nil, errNotPak
	}
	listAt := int64(h.FileListOffset)
	var counts [2]uint32
	if err := binary.Read(io.NewSectionReader(r, listAt, 8), binary.LittleEndian, &counts); err != nil {
		return nil, err
	}
	n, packed := counts[0], counts[1]
	if n > maxEntries || packed > n*entrySize+maxMeta {
		return nil, errNotPak
	}
	compressed := make([]byte, packed)
	if _, err := r.ReadAt(compressed, listAt+8); err != nil {
		return nil, err
	}
	list := make([]byte, int(n)*entrySize)
	got, err := lz4.UncompressBlock(compressed, list)
	if err != nil || got != len(list) {
		return nil, errNotPak
	}
	out := make([]pakEntry, 0, n)
	for i := range int(n) {
		b := list[i*entrySize : (i+1)*entrySize]
		name, _, _ := bytes.Cut(b[:nameSize], []byte{0})
		le := binary.LittleEndian
		out = append(out, pakEntry{
			name:   strings.ReplaceAll(string(name), "\\", "/"),
			offset: int64(le.Uint32(b[256:])) | int64(le.Uint16(b[260:]))<<32,
			part:   b[262], flags: b[263], size: le.Uint32(b[264:]), uncompress: le.Uint32(b[268:]),
		})
	}
	return out, nil
}

func decompress(raw []byte, flags uint8, size int) ([]byte, error) {
	switch flags & 0x0f {
	case compressNone:
		return raw, nil
	case compressZlib:
		zr, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		return io.ReadAll(io.LimitReader(zr, maxMeta))
	case compressLZ4:
		out := make([]byte, size)
		n, err := lz4.UncompressBlock(raw, out)
		return out[:n], err
	}
	return nil, fmt.Errorf("unsupported pak compression %d", flags&0x0f)
}
