// Package dotnet reads which external members a compiled .NET assembly assigns, from its ECMA-335 metadata and IL.
package dotnet

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"runtime"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Metadata table numbers this reader looks into.
const (
	tModule      = 0x00
	tTypeRef     = 0x01
	tTypeDef     = 0x02
	tField       = 0x04
	tMethodDef   = 0x06
	tParam       = 0x08
	tMemberRef   = 0x0A
	tDeclSec     = 0x0E
	tStandAlone  = 0x11
	tEvent       = 0x14
	tProperty    = 0x17
	tModuleRef   = 0x1A
	tTypeSpec    = 0x1B
	tAssembly    = 0x20
	tAssemblyRef = 0x23
	tFile        = 0x26
	tExported    = 0x27
	tGenParam    = 0x2A
	tMethodSpec  = 0x2B
	tGPConstr    = 0x2C
	tableCount   = 0x2D
)

// Coded index kinds (ECMA-335 II.24.2.6), each listing the tables its tag selects; -1 is an unused tag.
const (
	cTypeDefOrRef = iota
	cHasConstant
	cHasCustomAttribute
	cHasFieldMarshal
	cHasDeclSecurity
	cMemberRefParent
	cHasSemantics
	cMethodDefOrRef
	cMemberForwarded
	cImplementation
	cCustomAttributeType
	cResolutionScope
	cTypeOrMethodDef
)

var codedTables = [...][]int{
	cTypeDefOrRef:        {tTypeDef, tTypeRef, tTypeSpec},
	cHasConstant:         {tField, tParam, tProperty},
	cHasCustomAttribute:  {tMethodDef, tField, tTypeRef, tTypeDef, tParam, 0x09, tMemberRef, tModule, tDeclSec, tProperty, tEvent, tStandAlone, tModuleRef, tTypeSpec, tAssembly, tAssemblyRef, tFile, tExported, 0x28, tGenParam, tGPConstr, tMethodSpec},
	cHasFieldMarshal:     {tField, tParam},
	cHasDeclSecurity:     {tTypeDef, tMethodDef, tAssembly},
	cMemberRefParent:     {tTypeDef, tTypeRef, tModuleRef, tMethodDef, tTypeSpec},
	cHasSemantics:        {tEvent, tProperty},
	cMethodDefOrRef:      {tMethodDef, tMemberRef},
	cMemberForwarded:     {tField, tMethodDef},
	cImplementation:      {tFile, tAssemblyRef, tExported},
	cCustomAttributeType: {-1, -1, tMethodDef, tMemberRef, -1},
	cResolutionScope:     {tModule, tModuleRef, tAssemblyRef, tTypeRef},
	cTypeOrMethodDef:     {tTypeDef, tMethodDef},
}

// A column is u16, u32, a heap index, a table index (tbl+n) or a coded index (cod+kind).
const (
	u16 = iota
	u32
	str
	guid
	blob
	tbl = 100
	cod = 200
)

var schema = [tableCount][]int{
	0x00: {u16, str, guid, guid, guid},
	0x01: {cod + cResolutionScope, str, str},
	0x02: {u32, str, str, cod + cTypeDefOrRef, tbl + tField, tbl + tMethodDef},
	0x03: {tbl + tField},
	0x04: {u16, str, blob},
	0x05: {tbl + tMethodDef},
	0x06: {u32, u16, u16, str, blob, tbl + tParam},
	0x07: {tbl + tParam},
	0x08: {u16, u16, str},
	0x09: {tbl + tTypeDef, cod + cTypeDefOrRef},
	0x0A: {cod + cMemberRefParent, str, blob},
	0x0B: {u16, cod + cHasConstant, blob},
	0x0C: {cod + cHasCustomAttribute, cod + cCustomAttributeType, blob},
	0x0D: {cod + cHasFieldMarshal, blob},
	0x0E: {u16, cod + cHasDeclSecurity, blob},
	0x0F: {u16, u32, tbl + tTypeDef},
	0x10: {u32, tbl + tField},
	0x11: {blob},
	0x12: {tbl + tTypeDef, tbl + tEvent},
	0x13: {tbl + tEvent},
	0x14: {u16, str, cod + cTypeDefOrRef},
	0x15: {tbl + tTypeDef, tbl + tProperty},
	0x16: {tbl + tProperty},
	0x17: {u16, str, blob},
	0x18: {u16, tbl + tMethodDef, cod + cHasSemantics},
	0x19: {tbl + tTypeDef, cod + cMethodDefOrRef, cod + cMethodDefOrRef},
	0x1A: {str},
	0x1B: {blob},
	0x1C: {u16, cod + cMemberForwarded, str, tbl + tModuleRef},
	0x1D: {u32, tbl + tField},
	0x1E: {u32, u32},
	0x1F: {u32},
	0x20: {u32, u16, u16, u16, u16, u32, blob, str, str},
	0x21: {u32},
	0x22: {u32, u32, u32},
	0x23: {u16, u16, u16, u16, u32, blob, str, str, blob},
	0x24: {u32, tbl + tAssemblyRef},
	0x25: {u32, u32, u32, tbl + tAssemblyRef},
	0x26: {u32, str, blob},
	0x27: {u32, u32, str, str, cod + cImplementation},
	0x28: {u32, u32, str, cod + cImplementation},
	0x29: {tbl + tTypeDef, tbl + tTypeDef},
	0x2A: {u16, u16, cod + cTypeOrMethodDef, str},
	0x2B: {cod + cMethodDefOrRef, blob},
	0x2C: {tbl + tGenParam, cod + cTypeDefOrRef},
}

type module struct {
	img      []byte
	tables   []byte
	sections []*pe.Section
	strings  []byte
	blobs    []byte
	rows     [tableCount]int
	start    [tableCount]int
	rowSize  [tableCount]int
	cols     [tableCount][]int // column byte offsets, plus the row size at the end
	widths   [tableCount][]int
	heapWide [3]bool
}

// Writes lists the members of types in namespace game (or beneath it) that the assembly's method bodies assign, as
// "Namespace.Type::Member" with any set_ prefix removed: property setter calls and field stores through member
// references. A set_Value call on a type in namespace wrapper (a field wrapper such as Netcode's NetInt) counts as
// assigning the game member that loaded the wrapper.
func Writes(path, game, wrapper string) (out []string, err error) {
	data, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); !ok {
				panic(r)
			}
			out, err = nil, fmt.Errorf("%s: malformed assembly: %v", path, r)
		}
	}()
	m, err := open(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s := scan{m: m, game: game, wrapper: wrapper, written: map[string]bool{}, refs: map[uint32]ref{}}
	for i := 1; i <= m.rows[tMethodDef]; i++ {
		if rva := m.cell(tMethodDef, i, 0); rva != 0 {
			s.body(m.at(int(rva)))
		}
	}
	for member := range s.written {
		out = append(out, member)
	}
	slices.Sort(out)
	return out, nil
}

func open(data []byte) (*module, error) {
	f, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var dirs []pe.DataDirectory
	switch h := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		dirs = h.DataDirectory[:h.NumberOfRvaAndSizes]
	case *pe.OptionalHeader64:
		dirs = h.DataDirectory[:h.NumberOfRvaAndSizes]
	}
	if len(dirs) <= pe.IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR || dirs[pe.IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].VirtualAddress == 0 {
		return nil, errors.New("not a .NET assembly")
	}
	m := &module{img: data, sections: f.Sections}
	cli := m.at(int(dirs[pe.IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR].VirtualAddress))
	root := m.at(int(le32(cli[8:])))
	if le32(root) != 0x424A5342 {
		return nil, errors.New("no metadata root")
	}
	p := 16 + int(le32(root[12:]))
	streams := int(le16(root[p+2:]))
	p += 4
	var tables []byte
	for range streams {
		off, size := int(le32(root[p:])), int(le32(root[p+4:]))
		name := root[p+8:]
		name = name[:bytes.IndexByte(name, 0)]
		p += 8 + (len(name)+4)&^3
		body := root[off : off+size]
		switch string(name) {
		case "#~", "#-":
			tables = body
		case "#Strings":
			m.strings = body
		case "#Blob":
			m.blobs = body
		}
	}
	if tables == nil {
		return nil, errors.New("no metadata tables")
	}
	return m, m.layout(tables)
}

func (m *module) layout(t []byte) error {
	heaps := t[6]
	for i := range m.heapWide {
		m.heapWide[i] = heaps&(1<<i) != 0
	}
	valid := binary.LittleEndian.Uint64(t[8:])
	if valid>>tableCount != 0 {
		return errors.New("unsupported metadata tables")
	}
	p := 24
	for i := range tableCount {
		if valid&(1<<i) != 0 {
			m.rows[i] = int(le32(t[p:]))
			p += 4
		}
	}
	// Edit-and-continue images carry an extra 4 bytes after the row counts.
	if heaps&0x40 != 0 {
		p += 4
	}
	for i := range tableCount {
		offsets := []int{0}
		var widths []int
		for _, c := range schema[i] {
			w := m.width(c)
			widths = append(widths, w)
			offsets = append(offsets, offsets[len(offsets)-1]+w)
		}
		m.cols[i], m.widths[i] = offsets, widths
		m.rowSize[i] = offsets[len(offsets)-1]
		m.start[i] = p
		p += m.rowSize[i] * m.rows[i]
	}
	if p > len(t) {
		return errors.New("metadata tables are truncated")
	}
	m.tables = t
	return nil
}

func (m *module) width(c int) int {
	switch {
	case c == u16:
		return 2
	case c == u32:
		return 4
	case c == str:
		return wide(m.heapWide[0])
	case c == guid:
		return wide(m.heapWide[1])
	case c == blob:
		return wide(m.heapWide[2])
	case c >= cod:
		targets := codedTables[c-cod]
		tag := bits.Len(uint(len(targets) - 1))
		most := 0
		for _, t := range targets {
			if t >= 0 {
				most = max(most, m.rows[t])
			}
		}
		return wide(most >= 1<<(16-tag))
	default:
		return wide(m.rows[c-tbl] >= 1<<16)
	}
}

func wide(b bool) int {
	if b {
		return 4
	}
	return 2
}

// at returns the image from rva on, or nil when no section holds it.
func (m *module) at(rva int) []byte {
	for _, s := range m.sections {
		if rva >= int(s.VirtualAddress) && rva < int(s.VirtualAddress)+int(max(s.VirtualSize, s.Size)) {
			return m.img[rva-int(s.VirtualAddress)+int(s.Offset):]
		}
	}
	return nil
}

// cell reads column c of the 1-based row in table t.
func (m *module) cell(t, row, c int) uint32 {
	p := m.start[t] + (row-1)*m.rowSize[t] + m.cols[t][c]
	if m.widths[t][c] == 2 {
		return uint32(le16(m.tables[p:]))
	}
	return le32(m.tables[p:])
}

func (m *module) str(i uint32) string {
	s := m.strings[i:]
	return string(s[:bytes.IndexByte(s, 0)])
}

func (m *module) blob(i uint32) []byte {
	n, size := compressed(m.blobs[i:])
	return m.blobs[int(i)+size : int(i)+size+int(n)]
}

// compressed decodes an ECMA-335 compressed unsigned integer, returning it and its byte length.
func compressed(b []byte) (uint32, int) {
	switch {
	case b[0]&0x80 == 0:
		return uint32(b[0]), 1
	case b[0]&0xC0 == 0x80:
		return uint32(b[0]&0x3F)<<8 | uint32(b[1]), 2
	default:
		return uint32(b[0]&0x1F)<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), 4
	}
}

func decode(kind int, v uint32) (table, row int) {
	n := len(codedTables[kind])
	tag := bits.Len(uint(n - 1))
	return codedTables[kind][v&(1<<tag-1)], int(v >> tag)
}

// typeRef names a referenced type as its namespace and "Outer+Inner" name; nested references take the outermost
// type's namespace.
func (m *module) typeRef(row int) (ns, name string) {
	name = m.str(m.cell(tTypeRef, row, 1))
	scope, parent := decode(cResolutionScope, m.cell(tTypeRef, row, 0))
	if scope == tTypeRef && parent > 0 && parent != row {
		outerNS, outer := m.typeRef(parent)
		return outerNS, outer + "+" + name
	}
	return m.str(m.cell(tTypeRef, row, 2)), name
}

// typeOf names the referenced type a TypeDefOrRef-coded table/row points at, looking through a generic
// instantiation; the module's own types have no name here.
func (m *module) typeOf(table, row int) (ns, name string) {
	switch table {
	case tTypeRef:
		return m.typeRef(row)
	case tTypeSpec:
		sig := m.blob(m.cell(tTypeSpec, row, 0))
		if len(sig) > 2 && sig[0] == 0x15 {
			return m.sigType(sig[1:])
		}
	}
	return "", ""
}

// sigType names the class or value type at the start of a signature type, skipping custom modifiers.
func (m *module) sigType(sig []byte) (ns, name string) {
	for len(sig) > 1 && (sig[0] == 0x1F || sig[0] == 0x20) {
		_, n := compressed(sig[1:])
		sig = sig[1+n:]
	}
	if len(sig) > 1 && sig[0] == 0x15 {
		sig = sig[1:]
	}
	if len(sig) < 2 || (sig[0] != 0x12 && sig[0] != 0x11) {
		return "", ""
	}
	v, _ := compressed(sig[1:])
	table, row := decode(cTypeDefOrRef, v)
	return m.typeOf(table, row)
}

// ref is a resolved MemberRef: its declaring type's namespace and full name, its name, and the namespace of the
// field's type or the method's return type.
type ref struct {
	ns, owner, name, typeNS string
	ok                      bool
}

type scan struct {
	m             *module
	game, wrapper string
	written       map[string]bool
	refs          map[uint32]ref
}

func (s *scan) ref(token uint32) ref {
	if token>>24 != tMemberRef {
		return ref{}
	}
	if r, ok := s.refs[token]; ok {
		return r
	}
	m, row := s.m, int(token&0xFFFFFF)
	table, parent := decode(cMemberRefParent, m.cell(tMemberRef, row, 0))
	ns, owner := m.typeOf(table, parent)
	r := ref{ns: ns, owner: ns + "." + owner, name: m.str(m.cell(tMemberRef, row, 1)), ok: owner != ""}
	if sig := m.blob(m.cell(tMemberRef, row, 2)); len(sig) > 0 {
		if sig[0] == 0x06 {
			r.typeNS, _ = m.sigType(sig[1:])
		} else {
			p := 1
			if sig[0]&0x10 != 0 {
				_, n := compressed(sig[p:])
				p += n
			}
			_, n := compressed(sig[p:])
			r.typeNS, _ = m.sigType(sig[p+n:])
		}
	}
	s.refs[token] = r
	return r
}

func (s *scan) inGame(ns string) bool {
	return ns == s.game || strings.HasPrefix(ns, s.game+".")
}

// body walks one method's IL. Wrapper-typed game members are kept on a small stack so that a later set_Value on the
// wrapper is charged to the member that loaded it; any other call on a wrapper type consumes one.
// ponytail: approximates the evaluation stack; a wrapper passed to a non-wrapper method stays on it until the method
// ends, so a full stack simulation would be the upgrade if that misattributes in practice.
func (s *scan) body(b []byte) {
	var code []byte
	switch b[0] & 3 {
	case 2:
		code = b[1 : 1+int(b[0]>>2)]
	case 3:
		header := int(b[1]>>4) * 4
		code = b[header : header+int(le32(b[4:]))]
	default:
		return
	}
	var loaded []string
	pushed := false
	for p := 0; p < len(code); {
		op := int(code[p])
		p++
		if op == 0xFE {
			op = 0xFE00 | int(code[p])
			p++
		}
		size, ok := operand(op)
		if !ok {
			return
		}
		if op == 0x45 {
			size = 4 + 4*int(le32(code[p:]))
		}
		var token uint32
		if size == 4 {
			token = le32(code[p:])
		}
		p += size
		justPushed := false
		switch op {
		case 0x28, 0x6F: // call, callvirt
			r := s.ref(token)
			switch {
			case !r.ok:
			case s.inGame(r.ns) && strings.HasPrefix(r.name, "set_"):
				s.written[r.owner+"::"+r.name[4:]] = true
			case s.inGame(r.ns) && strings.HasPrefix(r.name, "get_") && r.typeNS == s.wrapper:
				loaded = append(loaded, r.owner+"::"+r.name[4:])
				justPushed = true
			case r.ns == s.wrapper && len(loaded) > 0:
				if r.name == "set_Value" {
					s.written[loaded[len(loaded)-1]] = true
				}
				loaded = loaded[:len(loaded)-1]
			}
		case 0x7B, 0x7C, 0x7E, 0x7F: // ldfld, ldflda, ldsfld, ldsflda
			if r := s.ref(token); r.ok && s.inGame(r.ns) && r.typeNS == s.wrapper {
				loaded = append(loaded, r.owner+"::"+r.name)
				justPushed = true
			}
		case 0x7D, 0x80: // stfld, stsfld
			if r := s.ref(token); r.ok && s.inGame(r.ns) {
				s.written[r.owner+"::"+r.name] = true
			}
		case 0x25: // dup, as compound assignments do to read the wrapper before setting it
			if pushed {
				loaded = append(loaded, loaded[len(loaded)-1])
				justPushed = true
			}
		case 0x2A: // ret
			loaded = loaded[:0]
		}
		pushed = justPushed
	}
}

// operand is the operand size of an IL opcode (0xFExx for two-byte opcodes); switch's is variable and handled by
// the caller.
func operand(op int) (int, bool) {
	switch {
	case op <= 0x0D, op == 0x14, op >= 0x15 && op <= 0x1E, op == 0x25, op == 0x26, op == 0x2A,
		op >= 0x46 && op <= 0x6E, op == 0x76, op == 0x7A, op >= 0x82 && op <= 0x8B, op == 0x8E,
		op >= 0x90 && op <= 0xA2, op >= 0xB3 && op <= 0xBA, op == 0xC3, op >= 0xD1 && op <= 0xDC,
		op == 0xDF, op == 0xE0:
		return 0, true
	case op >= 0x0E && op <= 0x13, op == 0x1F, op >= 0x2B && op <= 0x37, op == 0xDE:
		return 1, true
	case op == 0x20, op == 0x22, op >= 0x27 && op <= 0x29, op >= 0x38 && op <= 0x44, op >= 0x6F && op <= 0x75,
		op == 0x79, op >= 0x7B && op <= 0x81, op == 0x8C, op == 0x8D, op == 0x8F, op >= 0xA3 && op <= 0xA5,
		op == 0xC2, op == 0xC6, op == 0xD0, op == 0xDD:
		return 4, true
	case op == 0x21, op == 0x23:
		return 8, true
	case op == 0x45:
		return 0, true
	case op >= 0xFE00 && op <= 0xFE05, op == 0xFE0F, op == 0xFE11, op == 0xFE13, op == 0xFE14, op == 0xFE17,
		op == 0xFE18, op == 0xFE1A, op == 0xFE1D, op == 0xFE1E:
		return 0, true
	case op == 0xFE12, op == 0xFE19:
		return 1, true
	case op >= 0xFE09 && op <= 0xFE0E:
		return 2, true
	case op == 0xFE06, op == 0xFE07, op == 0xFE15, op == 0xFE16, op == 0xFE1C:
		return 4, true
	}
	return 0, false
}

func le16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func le32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
