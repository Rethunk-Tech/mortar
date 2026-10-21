package symsrv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	msfMagic = "Microsoft C/C++ MSF 7.00\r\n\x1aDS\x00\x00\x00"
	// maxDirectoryBytes and maxStreamBytes bound what a crafted PDB can make Lookup allocate, beside the file's size.
	maxDirectoryBytes = 16 << 20
	maxStreamBytes    = 256 << 20
	dbiStream         = 3
	dbiHeaderSize     = 64
	sectionHeaderSize = 40
	// sectionHeadersSlot is the index of the section header stream in the DBI's optional debug header.
	sectionHeadersSlot = 5
	sPub32             = 0x110E
	pubFunction        = 0x2
)

var errNotPDB = errors.New("not a Windows PDB")

// msf reads streams of a Multi-Stream File, the container of a Windows PDB: fixed-size blocks, and a directory listing
// each stream's size and blocks.
type msf struct {
	r         io.ReaderAt
	size      int64
	block     int64
	sizes     []uint32
	blocks    [][]uint32
	numBlocks int64
}

func openMSF(r io.ReaderAt, size int64) (*msf, error) {
	head := make([]byte, 56)
	if _, err := r.ReadAt(head, 0); err != nil || string(head[:32]) != msfMagic {
		return nil, errNotPDB
	}
	le := binary.LittleEndian
	m := &msf{r: r, size: size, block: int64(le.Uint32(head[32:])), numBlocks: int64(le.Uint32(head[40:]))}
	if m.block != 512 && m.block != 1024 && m.block != 2048 && m.block != 4096 || m.numBlocks*m.block > size {
		return nil, errNotPDB
	}
	dirBytes := int64(le.Uint32(head[44:]))
	if dirBytes < 4 || dirBytes > maxDirectoryBytes || dirBytes > size {
		return nil, errNotPDB
	}
	mapBlocks := (dirBytes + m.block - 1) / m.block
	blockMap, err := m.readBlock(int64(le.Uint32(head[52:])), mapBlocks*4)
	if err != nil {
		return nil, err
	}
	indices := make([]uint32, mapBlocks)
	for i := range indices {
		indices[i] = le.Uint32(blockMap[i*4:])
	}
	dir, err := m.read(indices, dirBytes)
	if err != nil {
		return nil, err
	}
	n := int64(le.Uint32(dir))
	if 4+n*4 > dirBytes {
		return nil, errNotPDB
	}
	p := 4 + n*4
	for i := range n {
		size := le.Uint32(dir[4+i*4:])
		if size == 0xFFFFFFFF {
			size = 0
		}
		count := (int64(size) + m.block - 1) / m.block
		if p+count*4 > dirBytes {
			return nil, errNotPDB
		}
		list := make([]uint32, count)
		for j := range list {
			list[j] = le.Uint32(dir[p+int64(j)*4:])
		}
		p += count * 4
		m.sizes, m.blocks = append(m.sizes, size), append(m.blocks, list)
	}
	return m, nil
}

func (m *msf) readBlock(index, n int64) ([]byte, error) {
	if index >= m.numBlocks || n > m.block {
		return nil, errNotPDB
	}
	b := make([]byte, n)
	_, err := m.r.ReadAt(b, index*m.block)
	return b, err
}

// read joins the first n bytes of the listed blocks.
func (m *msf) read(blocks []uint32, n int64) ([]byte, error) {
	out := make([]byte, 0, n)
	for _, b := range blocks {
		chunk, err := m.readBlock(int64(b), min(m.block, n-int64(len(out))))
		if err != nil {
			return nil, err
		}
		out = append(out, chunk...)
	}
	return out, nil
}

func (m *msf) stream(i int) ([]byte, error) {
	// A stream larger than the file can only repeat blocks.
	if i < 0 || i >= len(m.sizes) || m.sizes[i] > maxStreamBytes || int64(m.sizes[i]) > m.size {
		return nil, fmt.Errorf("PDB stream %d is missing or too large", i)
	}
	return m.read(m.blocks[i], int64(m.sizes[i]))
}

// Lookup names the public function symbol nearest at or below rva in the Windows PDB r holds, undecorated, and rva's
// offset into it.
func Lookup(r io.ReaderAt, size int64, rva uint64) (name string, offset uint64, err error) {
	m, err := openMSF(r, size)
	if err != nil {
		return "", 0, err
	}
	dbi, err := m.stream(dbiStream)
	if err != nil {
		return "", 0, err
	}
	if len(dbi) < dbiHeaderSize {
		return "", 0, errNotPDB
	}
	le := binary.LittleEndian
	substreams := int64(dbiHeaderSize)
	for _, at := range []int{24, 28, 32, 36, 40, 52} {
		// Each size is signed; read unsigned, a negative one is too large for the stream and is refused below.
		substreams += int64(le.Uint32(dbi[at:]))
	}
	slot := substreams + sectionHeadersSlot*2
	if slot+2 > int64(len(dbi)) {
		return "", 0, errNotPDB
	}
	headers, err := m.stream(int(le.Uint16(dbi[slot:])))
	if err != nil {
		return "", 0, err
	}
	var sections []uint64
	for h := headers; len(h) >= sectionHeaderSize; h = h[sectionHeaderSize:] {
		sections = append(sections, uint64(le.Uint32(h[12:])))
	}
	records, err := m.stream(int(le.Uint16(dbi[20:])))
	if err != nil {
		return "", 0, err
	}
	var best []byte
	var bestAt uint64
	for p := 0; p+4 <= len(records); {
		length := int(le.Uint16(records[p:]))
		next := p + 2 + length
		if length < 2 || next > len(records) {
			break
		}
		rec := records[p+2 : next]
		p = next
		if le.Uint16(rec) != sPub32 || len(rec) < 13 || le.Uint32(rec[2:])&pubFunction == 0 {
			continue
		}
		seg := int(le.Uint16(rec[10:]))
		if seg < 1 || seg > len(sections) {
			continue
		}
		at := sections[seg-1] + uint64(le.Uint32(rec[6:]))
		if at <= rva && (best == nil || at > bestAt) {
			best, bestAt = rec[12:], at
		}
	}
	if best == nil {
		return "", 0, errors.New("no public symbol at or below the address")
	}
	raw, _, _ := bytes.Cut(best, []byte{0})
	return undecorate(string(raw)), rva - bestAt, nil
}

// undecorate turns an MSVC-decorated name into "Scope::Name": "?DrawBuffers@GfxDeviceD3D11@@QEAAHPEAH@Z" is
// GfxDeviceD3D11::DrawBuffers, ??0 and ??1 a constructor and destructor. Any other form, a plain C name among them,
// is kept as it is.
func undecorate(s string) string {
	rest, ok := strings.CutPrefix(s, "?")
	if !ok {
		return s
	}
	special := ""
	switch {
	case strings.HasPrefix(rest, "?0"):
		special, rest = "ctor", rest[2:]
	case strings.HasPrefix(rest, "?1"):
		special, rest = "dtor", rest[2:]
	case strings.HasPrefix(rest, "?"):
		return s
	}
	qualified, _, ok := strings.Cut(rest, "@@")
	if !ok || qualified == "" {
		return s
	}
	parts := strings.Split(qualified, "@")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	switch special {
	case "ctor":
		parts = append(parts, parts[len(parts)-1])
	case "dtor":
		parts = append(parts, "~"+parts[len(parts)-1])
	}
	return strings.Join(parts, "::")
}
