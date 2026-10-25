package dotnet

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Member is one method an assembly defines: its type as a stack frame names it ("Namespace.Outer+Inner"), and its
// parameters' names in order.
type Member struct {
	Type   string `json:"type"`
	Method string `json:"method"`
	Static bool   `json:"static"`
	// Void is set when the method returns nothing.
	Void   bool     `json:"void"`
	Params []string `json:"params"`
}

const methodStatic = 0x10

// Members lists the methods of every type the assembly at path defines, skipping compiler-generated types and methods
// (names holding '<'), in table order.
func Members(path string) (out []Member, err error) {
	data, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer malformed(path, &out, &err)
	m, err := open(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	outer := map[int]int{}
	for row := 1; row <= m.rows[tNested]; row++ {
		outer[int(m.cell(tNested, row, 0))] = int(m.cell(tNested, row, 1))
	}
	for t := 1; t <= m.rows[tTypeDef]; t++ {
		typ := m.nestedName(t, outer)
		if strings.Contains(typ, "<") {
			continue
		}
		last := m.rows[tMethodDef] + 1
		if t < m.rows[tTypeDef] {
			last = int(m.cell(tTypeDef, t+1, 5))
		}
		for md := int(m.cell(tTypeDef, t, 5)); md < last && md <= m.rows[tMethodDef]; md++ {
			name := m.str(m.cell(tMethodDef, md, 3))
			if strings.Contains(name, "<") {
				continue
			}
			out = append(out, Member{
				Type:   typ,
				Method: name,
				Static: m.cell(tMethodDef, md, 2)&methodStatic != 0,
				Void:   returnsVoid(m.blob(m.cell(tMethodDef, md, 4))),
				Params: m.paramNames(md),
			})
		}
	}
	return out, nil
}

// returnsVoid reads a method signature's return type: after the calling convention, a generic method's type
// parameter count and the parameter count.
func returnsVoid(sig []byte) bool {
	if len(sig) < 2 {
		return false
	}
	p := 1
	if sig[0]&0x10 != 0 {
		_, n := compressed(sig[p:])
		p += n
	}
	_, n := compressed(sig[p:])
	p += n
	return p < len(sig) && sig[p] == 0x01
}

// nestedName is typeDefName with the nesting read once for the whole module.
func (m *module) nestedName(t int, outer map[int]int) string {
	name := m.str(m.cell(tTypeDef, t, 1))
	for range 32 {
		o := outer[t]
		if o == 0 || o == t {
			break
		}
		t = o
		name = m.str(m.cell(tTypeDef, t, 1)) + "+" + name
	}
	if ns := m.str(m.cell(tTypeDef, t, 2)); ns != "" {
		return ns + "." + name
	}
	return name
}

// paramNames are a method's parameter names in order; sequence 0 is the return value's row, when it has one.
func (m *module) paramNames(md int) []string {
	last := m.rows[tParam] + 1
	if md < m.rows[tMethodDef] {
		last = int(m.cell(tMethodDef, md+1, 5))
	}
	names := []string{}
	for p := int(m.cell(tMethodDef, md, 5)); p < last && p <= m.rows[tParam]; p++ {
		if m.cell(tParam, p, 1) != 0 {
			names = append(names, m.str(m.cell(tParam, p, 2)))
		}
	}
	return names
}
