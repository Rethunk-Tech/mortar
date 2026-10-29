package dotnet

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

const (
	// maxPDBBytes bounds a PDB read or inflated; a mod's symbols are kilobytes to a few megabytes.
	maxPDBBytes      = 64 << 20
	embeddedPDBMagic = 0x4244504D // "MPDB"
	codeViewMagic    = 0x53445352 // "RSDS"
	tDocument        = 0x30
	tMethodDebugInfo = 0x31
)

// SourceLine is the source file's base name and line ("Patch.cs:17") of IL offset il in method ("Namespace.Type::Method",
// taking params arguments, or -1 when unknown) of the assembly at path, read from its portable PDB, embedded or beside
// it. It is "" when the assembly has none, when the PDB is not the one built with this DLL (a weaver rewrote the DLL
// and left the PDB), when the name matches no method or several overloads, and when the method has no sequence point
// at or before il.
func SourceLine(path, method string, params, il int) string {
	s, _ := sourceLine(path, method, params, il)
	return s
}

func sourceLine(path, method string, params, il int) (source string, err error) {
	data, err := readAssembly(path)
	if err != nil {
		return "", err
	}
	defer malformed(path, &source, &err)
	m, err := open(data)
	if err != nil {
		return "", err
	}
	row := m.methodRow(method, params)
	cv := m.debugEntry(debugCodeView)
	if row == 0 || len(cv) < 20 || le32(cv) != codeViewMagic {
		return "", nil
	}
	pdb, err := m.pdbBytes(path)
	if err != nil || pdb == nil {
		return "", err
	}
	return sequenceLine(pdb, cv[4:20], row, il)
}

// pdbBytes is the portable PDB the DLL embeds, inflated, or the one beside it.
func (m *module) pdbBytes(dll string) ([]byte, error) {
	if e := m.debugEntry(debugEmbeddedPDB); e != nil {
		if len(e) < 8 || le32(e) != embeddedPDBMagic {
			return nil, errors.New("malformed embedded PDB")
		}
		size := int64(le32(e[4:]))
		if size > maxPDBBytes {
			return nil, fmt.Errorf("embedded PDB claims %d bytes", size)
		}
		out, err := io.ReadAll(io.LimitReader(flate.NewReader(bytes.NewReader(e[8:])), size+1))
		if err != nil || int64(len(out)) != size {
			return nil, errors.New("embedded PDB does not inflate to its stated size")
		}
		return out, nil
	}
	path := strings.TrimSuffix(dll, filepath.Ext(dll)) + ".pdb"
	if siblingPDB(dll) != SymbolsPortable {
		return nil, nil
	}
	f, err := safefs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return safefs.ReadCapped(f, maxPDBBytes)
}

// methodRow is the MethodDef row of the one method named "Namespace.Type::Method" with params parameters (any count
// when params is negative), or 0.
func (m *module) methodRow(method string, params int) int {
	typ, name, ok := strings.Cut(method, "::")
	if !ok {
		return 0
	}
	found := 0
	for t := 1; t <= m.rows[tTypeDef]; t++ {
		if m.typeDefName(t) != typ {
			continue
		}
		for md := range m.methodsOf(t) {
			if m.str(m.cell(tMethodDef, md, 3)) != name || params >= 0 && paramCount(m.blob(m.cell(tMethodDef, md, 4))) != params {
				continue
			}
			if found != 0 {
				return 0
			}
			found = md
		}
	}
	return found
}

func paramCount(sig []byte) int {
	p := 1
	if sig[0]&0x10 != 0 {
		_, n := compressed(sig[p:])
		p += n
	}
	n, _ := compressed(sig[p:])
	return int(n)
}

// sequenceLine reads a portable PDB's MethodDebugInformation row (the same row as the method's MethodDef) and returns
// the document and line of the last visible sequence point at or before il. id is the DLL's CodeView GUID, which the
// PDB's own id must match.
func sequenceLine(pdb, id []byte, row, il int) (string, error) {
	if len(pdb) < 16 || le32(pdb) != 0x424A5342 {
		return "", errors.New("not a portable PDB")
	}
	p := 16 + int(le32(pdb[12:]))
	streams := int(le16(pdb[p+2:]))
	p += 4
	var tables, blobs, pdbStream []byte
	for range streams {
		off, size := int(le32(pdb[p:])), int(le32(pdb[p+4:]))
		name := pdb[p+8:]
		name = name[:bytes.IndexByte(name, 0)]
		p += 8 + (len(name)+4)&^3
		switch body := pdb[off : off+size]; string(name) {
		case "#~":
			tables = body
		case "#Blob":
			blobs = body
		case "#Pdb":
			pdbStream = body
		}
	}
	if !bytes.Equal(pdbStream[:16], id) {
		return "", errors.New("PDB was not built with this assembly")
	}
	valid := le64(tables[8:])
	if valid&(1<<tDocument-1) != 0 {
		return "", errors.New("PDB holds type system tables")
	}
	rows := map[int]int{}
	p = 24
	for t := range 64 {
		if valid&(1<<t) != 0 {
			rows[t] = int(le32(tables[p:]))
			p += 4
		}
	}
	if row > rows[tMethodDebugInfo] {
		return "", nil
	}
	guidW, blobW := wide(tables[6]&2 != 0), wide(tables[6]&4 != 0)
	docW := wide(rows[tDocument] >= 1<<16)
	start := p + rows[tDocument]*(2*blobW+2*guidW) + (row-1)*(docW+blobW)
	doc := int(widthRead(tables[start:], docW))
	points := widthRead(tables[start+docW:], blobW)
	if points == 0 {
		return "", nil
	}
	n, size := compressed(blobs[points:])
	doc, line := sequencePoint(blobs[int(points)+size:int(points)+size+int(n)], doc, il)
	if line == 0 || doc == 0 || doc > rows[tDocument] {
		return "", nil
	}
	docRow := p + (doc-1)*(2*blobW+2*guidW)
	return documentName(blobs, widthRead(tables[docRow:], blobW)) + fmt.Sprintf(":%d", line), nil
}

// sequencePoint walks a sequence points blob (Portable PDB spec, "Sequence Points Blob") and returns the document and
// start line of the last visible point at or before il, or line 0.
func sequencePoint(b []byte, doc, il int) (bestDoc, bestLine int) {
	_, n := compressed(b) // local signature
	b = b[n:]
	if doc == 0 {
		d, n := compressed(b)
		doc, b = int(d), b[n:]
	}
	offset, line, seen := 0, 0, false
	for i := 0; len(b) > 0; i++ {
		delta, n := compressed(b)
		b = b[n:]
		if i > 0 && delta == 0 {
			d, n := compressed(b)
			doc, b = int(d), b[n:]
			continue
		}
		offset += int(delta)
		dLines, n := compressed(b)
		b = b[n:]
		var dCols int
		if dLines == 0 {
			c, n := compressed(b)
			dCols, b = int(c), b[n:]
		} else {
			dCols, n = signedCompressed(b)
			b = b[n:]
		}
		if dLines == 0 && dCols == 0 {
			continue
		}
		if !seen {
			l, n := compressed(b)
			_, n2 := compressed(b[n:])
			line, b, seen = int(l), b[n+n2:], true
		} else {
			dl, n := signedCompressed(b)
			_, n2 := signedCompressed(b[n:])
			line, b = line+dl, b[n+n2:]
		}
		if offset > il {
			break
		}
		bestDoc, bestLine = doc, line
	}
	return bestDoc, bestLine
}

// signedCompressed decodes an ECMA-335 compressed signed integer: the unsigned form rotated right by one bit.
func signedCompressed(b []byte) (int, int) {
	u, n := compressed(b)
	v := int(u >> 1)
	if u&1 != 0 {
		v -= map[int]int{1: 0x40, 2: 0x2000, 4: 0x10000000}[n]
	}
	return v, n
}

// documentName is the base name of a Document row's name blob: a separator byte, then blob indices of its parts.
func documentName(blobs []byte, i uint32) string {
	n, size := compressed(blobs[i:])
	b := blobs[int(i)+size : int(i)+size+int(n)]
	var last []byte
	for b = b[1:]; len(b) > 0; {
		part, n := compressed(b)
		b = b[n:]
		if part != 0 {
			l, s := compressed(blobs[part:])
			last = blobs[int(part)+s : int(part)+s+int(l)]
		}
	}
	name := string(last)
	return name[strings.LastIndexAny(name, `/\`)+1:]
}

func widthRead(b []byte, w int) uint32 {
	if w == 2 {
		return uint32(le16(b))
	}
	return le32(b)
}

func le64(b []byte) uint64 { return uint64(le32(b)) | uint64(le32(b[4:]))<<32 }
