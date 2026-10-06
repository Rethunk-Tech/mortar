package dotnet

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Kinds of Patch.
const (
	// PatchSkip is a prefix that returns bool, so it can skip the original method.
	PatchSkip       = "prefix-skip"
	PatchPrefix     = "prefix"
	PatchPostfix    = "postfix"
	PatchTranspiler = "transpiler"
	PatchFinalizer  = "finalizer"
	// PatchHook is a MonoMod HookGen On hook, which wraps the original and can skip it like a skipping prefix.
	PatchHook = "hook"
)

// Patch is one Harmony patch an assembly declares with attributes: the method it targets as "Namespace.Type::Method"
// and its kind.
type Patch struct{ Target, Kind string }

// patchInfo is what [HarmonyPatch] attributes have said so far about a patch's target.
type patchInfo struct{ typ, method, methodType string }

func (p patchInfo) merge(q patchInfo) patchInfo {
	if q.typ != "" {
		p.typ = q.typ
	}
	if q.method != "" {
		p.method = q.method
	}
	if q.methodType != "" {
		p.methodType = q.methodType
	}
	return p
}

// target names the patched method; a getter, setter or constructor is named as the compiler names it.
func (p patchInfo) target() (string, bool) {
	method := p.method
	switch p.methodType {
	case "getter":
		method = "get_" + method
	case "setter":
		method = "set_" + method
	case "ctor":
		method = ".ctor"
	case "cctor":
		method = ".cctor"
	}
	if p.typ == "" || method == "" {
		return "", false
	}
	return p.typ + "::" + method, true
}

// HarmonyMethodType values (Harmony 2 and HarmonyX), as the attribute blob stores them.
var methodTypes = map[uint32]string{1: "getter", 2: "setter", 3: "ctor", 4: "cctor"}

var patchKinds = map[string]string{
	"HarmonyPrefix": PatchPrefix, "HarmonyPostfix": PatchPostfix, "HarmonyTranspiler": PatchTranspiler, "HarmonyFinalizer": PatchFinalizer,
	"Prefix": PatchPrefix, "Postfix": PatchPostfix, "Transpiler": PatchTranspiler, "Finalizer": PatchFinalizer,
}

// Patches lists the Harmony patches the assembly at path declares with [HarmonyPatch] on a class, its methods or both,
// each method a [HarmonyPrefix]-style attribute or its name (Prefix, Postfix, Transpiler, Finalizer) marks, and the
// HookGen hooks it subscribes: On.Type.add_Method as a hook, IL.Type.add_Method as a transpiler. Patches a plugin
// applies by calling Harmony.Patch or new Hook(...) at run time name their target only through reflection and are not
// seen.
func Patches(path string) (out []Patch, err error) {
	data, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer malformed(path, &out, &err)
	m, err := open(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	typeInfo := map[int]patchInfo{}
	methodInfo := map[int]patchInfo{}
	methodKind := map[int]string{}
	for row := 1; row <= m.rows[tCustomAttr]; row++ {
		kind, ctor := decode(cCustomAttributeType, m.cell(tCustomAttr, row, 1))
		if kind != tMemberRef {
			continue
		}
		parent, ref := decode(cMemberRefParent, m.cell(tMemberRef, ctor, 0))
		if parent != tTypeRef {
			continue
		}
		ns, name := m.typeRef(ref)
		if ns != "HarmonyLib" {
			continue
		}
		owner, def := decode(cHasCustomAttribute, m.cell(tCustomAttr, row, 0))
		switch {
		case name == "HarmonyPatch":
			info, ok := m.harmonyPatchArgs(m.blob(m.cell(tMemberRef, ctor, 2)), m.blob(m.cell(tCustomAttr, row, 2)))
			if !ok {
				continue
			}
			switch owner {
			case tTypeDef:
				typeInfo[def] = typeInfo[def].merge(info)
			case tMethodDef:
				methodInfo[def] = methodInfo[def].merge(info)
			}
		case patchKinds[name] != "" && owner == tMethodDef:
			methodKind[def] = patchKinds[name]
		}
	}
	seen := map[Patch]bool{}
	for t := 1; t <= m.rows[tTypeDef]; t++ {
		first := int(m.cell(tTypeDef, t, 5))
		last := m.rows[tMethodDef] + 1
		if t < m.rows[tTypeDef] {
			last = int(m.cell(tTypeDef, t+1, 5))
		}
		class, patchClass := typeInfo[t]
		for md := first; md < last && md <= m.rows[tMethodDef]; md++ {
			kind := methodKind[md]
			if kind == "" && patchClass {
				kind = patchKinds[m.str(m.cell(tMethodDef, md, 3))]
			}
			if kind == "" {
				continue
			}
			target, ok := class.merge(methodInfo[md]).target()
			if !ok {
				continue
			}
			if kind == PatchPrefix && returnsBool(m.blob(m.cell(tMethodDef, md, 4))) {
				kind = PatchSkip
			}
			if p := (Patch{Target: target, Kind: kind}); !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	for row := 1; row <= m.rows[tMemberRef]; row++ {
		parent, ref := decode(cMemberRefParent, m.cell(tMemberRef, row, 0))
		if parent != tTypeRef {
			continue
		}
		name := m.str(m.cell(tMemberRef, row, 1))
		ns, typ := m.typeRef(ref)
		hooked, isHook := strings.CutPrefix(name, "add_")
		root, gameNS, _ := strings.Cut(ns, ".")
		kind := map[string]string{"On": PatchHook, "IL": PatchTranspiler}[root]
		if !isHook || kind == "" {
			continue
		}
		if gameNS != "" {
			typ = gameNS + "." + typ
		}
		if p := (Patch{Target: typ + "::" + hooked, Kind: kind}); !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Patch) int { return strings.Compare(a.Target+a.Kind, b.Target+b.Kind) })
	return out, nil
}

// returnsBool reads a method signature's return type.
func returnsBool(sig []byte) bool {
	if len(sig) < 2 {
		return false
	}
	p := 1
	if sig[0]&0x10 != 0 {
		_, n := compressed(sig[p:])
		p += n
	}
	_, n := compressed(sig[p:])
	return p+n < len(sig) && sig[p+n] == 0x02
}

// harmonyPatchArgs reads a [HarmonyPatch] attribute's fixed arguments, typed by its constructor's signature: a Type is
// the declaring type, a first string the method name (or, followed by a second string, HarmonyX's type name then method
// name), a MethodType enum the kind of method. Argument type lists only pick an overload and are skipped.
func (m *module) harmonyPatchArgs(ctorSig, b []byte) (patchInfo, bool) {
	if len(b) < 2 || b[0] != 1 || b[1] != 0 || len(ctorSig) < 3 {
		return patchInfo{}, false
	}
	b = b[2:]
	count, n := compressed(ctorSig[1:])
	sig := ctorSig[1+n:]
	if len(sig) == 0 || sig[0] != 0x01 {
		return patchInfo{}, false
	}
	sig = sig[1:]
	var info patchInfo
	var strs []string
	for range count {
		if len(sig) == 0 {
			return patchInfo{}, false
		}
		switch sig[0] {
		case 0x0e:
			s, rest, ok := serString(b)
			if !ok {
				return patchInfo{}, false
			}
			strs, b, sig = append(strs, s), rest, sig[1:]
		case 0x12, 0x11:
			v, n := compressed(sig[1:])
			sig = sig[1+n:]
			ns, name := m.typeOf(decode(cTypeDefOrRef, v))
			switch {
			case ns == "System" && name == "Type":
				s, rest, ok := serString(b)
				if !ok {
					return patchInfo{}, false
				}
				info.typ, b = typeName(s), rest
			case len(b) >= 4:
				if name == "MethodType" {
					info.methodType = methodTypes[le32(b)]
				}
				b = b[4:]
			default:
				return patchInfo{}, false
			}
		default:
			// An array of argument types or ArgumentType values is last in every overload that has one.
			return info.withStrings(strs), true
		}
	}
	return info.withStrings(strs), true
}

func (p patchInfo) withStrings(strs []string) patchInfo {
	switch {
	case len(strs) >= 2 && p.typ == "":
		p.typ, p.method = typeName(strs[0]), strs[1]
	case len(strs) >= 1:
		p.method = strs[0]
	}
	return p
}

// typeName drops the assembly from a serialized type name ("Namespace.Type, Assembly-CSharp, Version=…").
func typeName(s string) string {
	name, _, _ := strings.Cut(s, ",")
	return strings.TrimSpace(name)
}
