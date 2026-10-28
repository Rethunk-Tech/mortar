package dotnet

import (
	"iter"
	"strings"
)

// Member is one method or field an assembly defines: its type as a stack frame names it ("Namespace.Outer+Inner"),
// and a method's parameters' names in order.
type Member struct {
	Type string `json:"type"`
	// Kind is "method" or "field".
	Kind   string `json:"kind"`
	Method string `json:"method"`
	Static bool   `json:"static"`
	// Void is set when the method returns nothing.
	Void   bool     `json:"void"`
	Params []string `json:"params"`
}

const (
	methodStatic = 0x10
	fieldStatic  = 0x10
)

// Members lists the fields and methods of every type the assembly at path defines, skipping compiler-generated types
// and members (names holding '<'), in table order.
func Members(path string) ([]Member, error) {
	return withModule(path, members)
}

func members(m *module) (out []Member, err error) {
	outer := map[int]int{}
	for row := 1; row <= m.rows[tNested]; row++ {
		outer[int(m.cell(tNested, row, 0))] = int(m.cell(tNested, row, 1))
	}
	for t := 1; t <= m.rows[tTypeDef]; t++ {
		typ := m.nestedName(t, outer)
		if strings.Contains(typ, "<") {
			continue
		}
		for f := range m.fieldsOf(t) {
			if name := m.str(m.cell(tField, f, 1)); !strings.Contains(name, "<") {
				out = append(out, Member{Type: typ, Kind: "field", Method: name, Static: m.cell(tField, f, 0)&fieldStatic != 0, Params: []string{}})
			}
		}
		for md := range m.methodsOf(t) {
			name := m.str(m.cell(tMethodDef, md, 3))
			if strings.Contains(name, "<") {
				continue
			}
			out = append(out, Member{
				Type:   typ,
				Kind:   "method",
				Method: name,
				Static: m.cell(tMethodDef, md, 2)&methodStatic != 0,
				Void:   returnType(m.blob(m.cell(tMethodDef, md, 4))) == elementVoid,
				Params: m.paramNames(md),
			})
		}
	}
	return out, nil
}

const (
	elementVoid = 0x01
	elementBool = 0x02
)

// returnType is the element type a method signature returns, read after the calling convention, a generic method's
// type parameter count and the parameter count; 0 when the signature is too short.
func returnType(sig []byte) byte {
	if len(sig) < 2 {
		return 0
	}
	p := 1
	if sig[0]&0x10 != 0 {
		_, n := compressed(sig[p:])
		p += n
	}
	_, n := compressed(sig[p:])
	if p += n; p < len(sig) {
		return sig[p]
	}
	return 0
}

// methodsOf is the MethodDef rows type t owns.
func (m *module) methodsOf(t int) iter.Seq[int] { return m.ownedRows(t, 5, tMethodDef) }

// fieldsOf is the Field rows type t owns.
func (m *module) fieldsOf(t int) iter.Seq[int] { return m.ownedRows(t, 4, tField) }

// ownedRows is the rows of table a TypeDef's list column col starts, up to where the next type's list starts.
func (m *module) ownedRows(t, col, table int) iter.Seq[int] {
	return func(yield func(int) bool) {
		last := m.rows[table] + 1
		if t < m.rows[tTypeDef] {
			last = int(m.cell(tTypeDef, t+1, col))
		}
		for row := int(m.cell(tTypeDef, t, col)); row < last && row <= m.rows[table]; row++ {
			if !yield(row) {
				return
			}
		}
	}
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
