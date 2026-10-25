package dotnet

import (
	"fmt"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// usageOps are the opcodes whose token names a method or field: call, callvirt, newobj, the field loads and stores,
// ldftn and ldvirtftn.
var usageOps = []int{0x28, 0x6F, 0x73, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F, 0x80, 0xFE06, 0xFE07}

// Usages lists the methods of the assembly at path whose code calls, constructs, loads or stores target, named
// "Namespace.Type::Member" as Members names it; each caller is "Type::Method", once, in table order.
func Usages(path, target string) (out []string, err error) {
	data, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer malformed(path, &out, &err)
	m, err := open(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	u := m.usageNames()
	out = []string{}
	for md := 1; md <= m.rows[tMethodDef]; md++ {
		rva := m.cell(tMethodDef, md, 0)
		if rva == 0 {
			continue
		}
		caller := u.methods[md]
		if !slices.Contains(out, caller) && slices.Contains(u.references(m, m.at(int(rva))), target) {
			out = append(out, caller)
		}
	}
	return out, nil
}

type usageNames struct {
	methods, fields []string
}

// usageNames names every method and field the module defines as "Type::Member", by row.
func (m *module) usageNames() usageNames {
	outer := map[int]int{}
	for row := 1; row <= m.rows[tNested]; row++ {
		outer[int(m.cell(tNested, row, 0))] = int(m.cell(tNested, row, 1))
	}
	u := usageNames{methods: make([]string, m.rows[tMethodDef]+1), fields: make([]string, m.rows[tField]+1)}
	for t := 1; t <= m.rows[tTypeDef]; t++ {
		typ := m.nestedName(t, outer)
		lastM, lastF := m.rows[tMethodDef]+1, m.rows[tField]+1
		if t < m.rows[tTypeDef] {
			lastM, lastF = int(m.cell(tTypeDef, t+1, 5)), int(m.cell(tTypeDef, t+1, 4))
		}
		for md := int(m.cell(tTypeDef, t, 5)); md < lastM && md <= m.rows[tMethodDef]; md++ {
			u.methods[md] = typ + "::" + m.str(m.cell(tMethodDef, md, 3))
		}
		for f := int(m.cell(tTypeDef, t, 4)); f < lastF && f <= m.rows[tField]; f++ {
			u.fields[f] = typ + "::" + m.str(m.cell(tField, f, 1))
		}
	}
	return u
}

// name is the member a method or field token names.
func (u usageNames) name(m *module, token uint32) string {
	row := int(token & 0xFFFFFF)
	switch token >> 24 {
	case tMethodDef:
		if row < len(u.methods) {
			return u.methods[row]
		}
	case tField:
		if row < len(u.fields) {
			return u.fields[row]
		}
	case tMemberRef:
		table, parent := decode(cMemberRefParent, m.cell(tMemberRef, row, 0))
		ns, owner := m.typeOf(table, parent)
		if owner == "" {
			return ""
		}
		if ns != "" {
			owner = ns + "." + owner
		}
		return owner + "::" + m.str(m.cell(tMemberRef, row, 1))
	case tMethodSpec:
		method := m.cell(tMethodSpec, row, 0)
		if method&1 == 0 {
			return u.name(m, tMethodDef<<24|method>>1)
		}
		return u.name(m, tMemberRef<<24|method>>1)
	}
	return ""
}

// references lists the members one method body's IL names.
func (u usageNames) references(m *module, b []byte) []string {
	var code []byte
	switch b[0] & 3 {
	case 2:
		code = b[1 : 1+int(b[0]>>2)]
	case 3:
		header := int(b[1]>>4) * 4
		code = b[header : header+int(le32(b[4:]))]
	default:
		return nil
	}
	var out []string
	for p := 0; p < len(code); {
		op := int(code[p])
		p++
		if op == 0xFE {
			op = 0xFE00 | int(code[p])
			p++
		}
		size, ok := operand(op)
		if !ok {
			return out
		}
		if op == 0x45 {
			size = 4 + 4*int(le32(code[p:]))
		}
		if size == 4 && slices.Contains(usageOps, op) {
			if n := u.name(m, le32(code[p:])); n != "" {
				out = append(out, n)
			}
		}
		p += size
	}
	return out
}
