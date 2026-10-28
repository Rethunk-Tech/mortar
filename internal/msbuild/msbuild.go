// Package msbuild edits MSBuild files (.csproj, .props, .targets) in place: it parses them to find each element's exact
// bytes, then changes only those, so the developer's formatting, comments and attribute order stay as they wrote them.
package msbuild

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

// Elem is one element of the file: its name, attributes, and where its bytes are.
type Elem struct {
	Name  string
	Attrs map[string]string
	// Start and End span the whole element; InnerStart and InnerEnd its content, equal for a self-closing one.
	Start, End           int
	InnerStart, InnerEnd int
	// Parent is the index of the enclosing element in the parse, -1 for the root.
	Parent int
	text   string
}

// Text is the element's content with XML entities resolved, empty when it holds child elements.
func (e Elem) Text() string {
	inner := e.text[e.InnerStart:e.InnerEnd]
	var out strings.Builder
	d := xml.NewDecoder(strings.NewReader("<x>" + inner + "</x>"))
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.CharData:
			out.Write(t)
		case xml.StartElement:
			if t.Name.Local != "x" {
				return ""
			}
		}
	}
	return out.String()
}

// Parse lists the elements of an MSBuild file in document order; comments and processing instructions are skipped.
func Parse(text string) ([]Elem, error) {
	d := xml.NewDecoder(strings.NewReader(text))
	var out []Elem
	var open []int
	for {
		start := int(d.InputOffset())
		tok, err := d.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			attrs := make(map[string]string, len(t.Attr))
			for _, a := range t.Attr {
				attrs[a.Name.Local] = a.Value
			}
			parent := -1
			if len(open) > 0 {
				parent = open[len(open)-1]
			}
			at := int(d.InputOffset())
			out = append(out, Elem{Name: t.Name.Local, Attrs: attrs, Start: start, InnerStart: at, InnerEnd: at, Parent: parent, text: text})
			open = append(open, len(out)-1)
		case xml.EndElement:
			if len(open) == 0 {
				return nil, fmt.Errorf("unexpected </%s>", t.Name.Local)
			}
			e := &out[open[len(open)-1]]
			open = open[:len(open)-1]
			e.End = int(d.InputOffset())
			// A self-closing element's end comes with no bytes of its own.
			if e.End != e.InnerStart {
				e.InnerEnd = start
			}
		}
	}
	if len(open) > 0 {
		return nil, errors.New("the file ends inside an element")
	}
	return out, nil
}

// Property is the first property called name, unescaped; empty when the file has none or does not parse.
func Property(text, name string) string {
	elems, err := Parse(text)
	if err != nil {
		return ""
	}
	if e, ok := property(elems, name); ok {
		return unescape.Replace(strings.TrimSpace(e.Text()))
	}
	return ""
}

func property(elems []Elem, name string) (Elem, bool) {
	for _, e := range elems {
		if strings.EqualFold(e.Name, name) && e.Parent >= 0 && elems[e.Parent].Name == "PropertyGroup" {
			return e, true
		}
	}
	return Elem{}, false
}

// SetProperty sets the first property called name to value, escaped for MSBuild and XML; a missing one is added to the
// first PropertyGroup, or to a new one.
func SetProperty(text, name, value string) (string, error) {
	return SetRawProperty(text, name, Escape(value))
}

// SetRawProperty is SetProperty for text already escaped, such as a list whose ; MSBuild must split on.
func SetRawProperty(text, name, raw string) (string, error) {
	elems, err := Parse(text)
	if err != nil {
		return text, err
	}
	if e, ok := property(elems, name); ok {
		if e.InnerStart == e.InnerEnd && e.End == e.InnerStart {
			return text[:e.Start] + "<" + name + ">" + raw + "</" + name + ">" + text[e.End:], nil
		}
		return text[:e.InnerStart] + raw + text[e.InnerEnd:], nil
	}
	line := "<" + name + ">" + raw + "</" + name + ">"
	for _, e := range elems {
		if e.Name == "PropertyGroup" && e.Parent == 0 {
			return appendChild(text, elems, e, line), nil
		}
	}
	return appendGroup(text, elems, "PropertyGroup", line)
}

// RemoveProperty drops every property called name, with its line.
func RemoveProperty(text, name string) (string, error) {
	return Remove(text, func(e Elem, parent string) bool { return e.Name == name && parent == "PropertyGroup" })
}

// Item is an item element: its type, its Include and its other attributes.
type Item struct {
	Type    string
	Include string
	Attrs   map[string]string
}

// Items lists the items of the given types in document order.
func Items(text string, types ...string) ([]Item, error) {
	elems, err := Parse(text)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, e := range elems {
		if slices.Contains(types, e.Name) && e.Parent >= 0 && elems[e.Parent].Name == "ItemGroup" {
			out = append(out, Item{Type: e.Name, Include: e.Attrs["Include"], Attrs: e.Attrs})
		}
	}
	return out, nil
}

// SetItem puts item in place of the first item whose type is one of same and whose Include matches; otherwise after the
// last item after picks, or in a new ItemGroup.
func SetItem(text string, item Item, same []string, after func(Item) bool) (string, error) {
	elems, err := Parse(text)
	if err != nil {
		return text, err
	}
	xmlText := item.xml()
	var last *Elem
	for i, e := range elems {
		if e.Parent < 0 || elems[e.Parent].Name != "ItemGroup" {
			continue
		}
		if slices.Contains(same, e.Name) && strings.EqualFold(e.Attrs["Include"], item.Include) {
			return text[:e.Start] + xmlText + text[e.End:], nil
		}
		if after(Item{Type: e.Name, Include: e.Attrs["Include"], Attrs: e.Attrs}) {
			last = &elems[i]
		}
	}
	if last != nil {
		// After the rest of its line, such as a trailing comment, unless that line also closes the group.
		at := last.End
		if le := strings.IndexByte(text[at:], '\n'); le >= 0 && !strings.Contains(text[at:at+le], "</") {
			at += le
		}
		return text[:at] + "\n" + indentOf(text, last.Start) + xmlText + text[at:], nil
	}
	return appendGroup(text, elems, "ItemGroup", xmlText)
}

// Remove drops every element match picks, given its parent's name, with the line it stands on when nothing else
// shares that line.
func Remove(text string, match func(e Elem, parent string) bool) (string, error) {
	elems, err := Parse(text)
	if err != nil {
		return text, err
	}
	var spans [][2]int
	for _, e := range elems {
		parent := ""
		if e.Parent >= 0 {
			parent = elems[e.Parent].Name
		}
		if match(e, parent) {
			spans = append(spans, lineSpan(text, e.Start, e.End))
		}
	}
	for _, span := range slices.Backward(spans) {
		text = text[:span[0]] + text[span[1]:]
	}
	return text, nil
}

// SetAttr sets attr to value on every element match picks, in place: an existing value's bytes are replaced, a missing
// attribute is added before the tag's end. It also serves .slnx solutions, which are XML.
func SetAttr(text string, match func(e Elem, parent string) bool, attr, value string) (string, error) {
	elems, err := Parse(text)
	if err != nil {
		return text, err
	}
	quoted := attr + `="` + escapeAttr(value) + `"`
	existing := regexp.MustCompile(`(\s)` + regexp.QuoteMeta(attr) + `\s*=\s*("[^"]*"|'[^']*')`)
	for _, e := range slices.Backward(elems) {
		parent := ""
		if e.Parent >= 0 {
			parent = elems[e.Parent].Name
		}
		if !match(e, parent) {
			continue
		}
		tag := text[e.Start:e.InnerStart]
		if loc := existing.FindStringSubmatchIndex(tag); loc != nil {
			tag = tag[:loc[3]] + quoted + tag[loc[1]:]
		} else {
			end := len(tag) - 1
			if strings.HasSuffix(tag, "/>") {
				end = len(tag) - 2
			}
			tag = strings.TrimRight(tag[:end], " \t") + " " + quoted + tag[end:]
			if strings.HasSuffix(tag, quoted+"/>") {
				tag = strings.TrimSuffix(tag, "/>") + " />"
			}
		}
		text = text[:e.Start] + tag + text[e.InnerStart:]
	}
	return text, nil
}

func (it Item) xml() string {
	var b strings.Builder
	b.WriteString("<" + it.Type + ` Include="` + escapeAttr(it.Include) + `"`)
	keys := make([]string, 0, len(it.Attrs))
	for k := range it.Attrs {
		if k != "Include" {
			keys = append(keys, k)
		}
	}
	// Version first, as the template writes it, then the rest in a fixed order.
	slices.SortFunc(keys, func(a, b string) int {
		if a == "Version" || b == "Version" {
			return map[bool]int{true: -1, false: 1}[a == "Version"]
		}
		return strings.Compare(a, b)
	})
	for _, k := range keys {
		b.WriteString(" " + k + `="` + escapeAttr(it.Attrs[k]) + `"`)
	}
	b.WriteString(" />")
	return b.String()
}

// lineSpan widens [start,end) to its whole line, break included, when only whitespace shares that line.
func lineSpan(text string, start, end int) [2]int {
	ls := strings.LastIndexByte(text[:start], '\n') + 1
	le := strings.IndexByte(text[end:], '\n')
	if le < 0 {
		le = len(text)
	} else {
		le += end + 1
	}
	if strings.TrimSpace(text[ls:start]) == "" && strings.TrimSpace(text[end:le]) == "" {
		return [2]int{ls, le}
	}
	return [2]int{start, end}
}

func indentOf(text string, at int) string {
	ls := strings.LastIndexByte(text[:at], '\n') + 1
	return text[ls : ls+len(text[ls:at])-len(strings.TrimLeft(text[ls:at], " \t"))]
}

// appendChild adds line as the last child of e, indented like its other children or one step past e.
func appendChild(text string, elems []Elem, e Elem, line string) string {
	indent := indentOf(text, e.Start) + "  "
	for _, c := range elems {
		if c.Start > e.Start && c.End <= e.End && elems[c.Parent].Start == e.Start {
			indent = indentOf(text, c.Start)
		}
	}
	at := e.InnerEnd
	ls := strings.LastIndexByte(text[:at], '\n') + 1
	if strings.TrimSpace(text[ls:at]) == "" && ls > e.InnerStart {
		return text[:ls] + indent + line + "\n" + text[ls:]
	}
	return text[:at] + "\n" + indent + line + "\n" + indentOf(text, e.Start) + text[at:]
}

// appendGroup adds a new group of kind holding line at the end of the root element.
func appendGroup(text string, elems []Elem, kind, line string) (string, error) {
	if len(elems) == 0 {
		return text, errors.New("the file has no root element")
	}
	root := elems[0]
	at := root.InnerEnd
	ls := strings.LastIndexByte(text[:at], '\n') + 1
	group := "  <" + kind + ">\n    " + line + "\n  </" + kind + ">\n"
	if strings.TrimSpace(text[ls:at]) == "" && ls > root.InnerStart {
		return text[:ls] + group + text[ls:], nil
	}
	return text[:at] + "\n" + group + text[at:], nil
}

// MSBuild expands $(), @(), %XX and splits on ; in a property's text, so the characters it reads specially are written
// as %XX. The XML escape runs last: escaping first would turn the ; of &lt; into %3B and break the entity.
var (
	escapeMSBuild = strings.NewReplacer("%", "%25", "$", "%24", "@", "%40", ";", "%3B", "'", "%27", "*", "%2A", "?", "%3F")
	unescape      = strings.NewReplacer("%25", "%", "%24", "$", "%40", "@", "%3B", ";", "%27", "'", "%2A", "*", "%3F", "?")
	escapeXML     = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
)

// Escape writes value as a property's text: MSBuild's special characters as %XX, then XML's.
func Escape(value string) string { return escapeXML.Replace(escapeMSBuild.Replace(value)) }

// EscapeXML escapes only what XML needs, for a value MSBuild should still read specially.
func EscapeXML(value string) string { return escapeXML.Replace(value) }

func escapeAttr(v string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(v)
}
