package fomod

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// Parse reads a ModuleConfig.xml document.
func Parse(data []byte) (Config, error) {
	var raw xmlConfig
	if err := xml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("fomod: %w", err)
	}
	cfg := Config{ModuleName: strings.TrimSpace(raw.ModuleName)}
	cfg.Required = appendOps(nil, raw.Required.Files, false)
	cfg.Required = appendOps(cfg.Required, raw.Required.Folders, true)
	for _, s := range raw.Steps.Steps {
		st := Step{Name: s.Name, Visible: s.Visible.Composite}
		for _, g := range s.Groups.Groups {
			gr := Group{Name: g.Name, Type: g.Type}
			for _, p := range g.Plugins.Plugins {
				pl := Plugin{
					Name:        p.Name,
					Description: strings.TrimSpace(p.Description),
					Image:       p.Image.Path,
					Flags:       flagSets(p.Flags.Flags),
					TypeName:    p.Type.Type.Name,
					DefaultType: p.Type.Dependency.Default.Name,
				}
				pl.Files = appendOps(pl.Files, p.Files.Files, false)
				pl.Files = appendOps(pl.Files, p.Files.Folders, true)
				for _, pat := range p.Type.Dependency.Patterns.Patterns {
					pl.Patterns = append(pl.Patterns, TypePattern{Type: pat.Dependencies.Composite, Name: pat.Type.Name})
				}
				gr.Plugins = append(gr.Plugins, pl)
			}
			st.Groups = append(st.Groups, gr)
		}
		cfg.Steps = append(cfg.Steps, st)
	}
	for _, p := range raw.Conditionals.Patterns.Patterns {
		c := Conditional{When: p.Dependencies.Composite}
		c.Files = appendOps(c.Files, p.Files.Files, false)
		c.Files = appendOps(c.Files, p.Files.Folders, true)
		cfg.Conditionals = append(cfg.Conditionals, c)
	}
	return cfg, nil
}

func flagSets(flags []xmlFlag) []FlagSet {
	out := make([]FlagSet, len(flags))
	for i, f := range flags {
		out[i] = FlagSet{Name: f.Name, Value: strings.TrimSpace(f.Value)}
	}
	return out
}

func appendOps(dst []Op, items []xmlFile, folder bool) []Op {
	for _, it := range items {
		dst = append(dst, Op{
			Source:      it.Source,
			Destination: it.Destination,
			Priority:    atoi(it.Priority),
			Folder:      folder,
		})
	}
	return dst
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

type xmlConfig struct {
	ModuleName   string          `xml:"moduleName"`
	Required     xmlFiles        `xml:"requiredInstallFiles"`
	Steps        xmlSteps        `xml:"installSteps"`
	Conditionals xmlConditionals `xml:"conditionalFileInstalls"`
}

type xmlSteps struct {
	Steps []xmlStep `xml:"installStep"`
}

type xmlStep struct {
	Name    string     `xml:"name,attr"`
	Visible xmlVisible `xml:"visible"`
	Groups  xmlGroups  `xml:"optionalFileGroups"`
}

type xmlVisible struct {
	Composite
}

type xmlGroups struct {
	Groups []xmlGroup `xml:"group"`
}

type xmlGroup struct {
	Name    string     `xml:"name,attr"`
	Type    string     `xml:"type,attr"`
	Plugins xmlPlugins `xml:"plugins"`
}

type xmlPlugins struct {
	Plugins []xmlPlugin `xml:"plugin"`
}

type xmlPlugin struct {
	Name        string      `xml:"name,attr"`
	Description string      `xml:"description"`
	Image       xmlImage    `xml:"image"`
	Files       xmlFiles    `xml:"files"`
	Flags       xmlFlags    `xml:"conditionFlags"`
	Type        xmlTypeDesc `xml:"typeDescriptor"`
}

type xmlImage struct {
	Path string `xml:"path,attr"`
}

type xmlFiles struct {
	Files   []xmlFile `xml:"file"`
	Folders []xmlFile `xml:"folder"`
}

type xmlFile struct {
	Source      string `xml:"source,attr"`
	Destination string `xml:"destination,attr"`
	Priority    string `xml:"priority,attr"`
}

type xmlFlags struct {
	Flags []xmlFlag `xml:"flag"`
}

type xmlFlag struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type xmlTypeDesc struct {
	Type       xmlTypeName `xml:"type"`
	Dependency xmlDepType  `xml:"dependencyType"`
}

type xmlTypeName struct {
	Name string `xml:"name,attr"`
}

type xmlDepType struct {
	Default  xmlTypeName `xml:"defaultType"`
	Patterns xmlTypePats `xml:"patterns"`
}

type xmlTypePats struct {
	Patterns []xmlTypePat `xml:"pattern"`
}

type xmlTypePat struct {
	Dependencies xmlDeps     `xml:"dependencies"`
	Type         xmlTypeName `xml:"type"`
}

type xmlConditionals struct {
	Patterns xmlCondPats `xml:"patterns"`
}

type xmlCondPats struct {
	Patterns []xmlCondPat `xml:"pattern"`
}

type xmlCondPat struct {
	Dependencies xmlDeps  `xml:"dependencies"`
	Files        xmlFiles `xml:"files"`
}

type xmlDeps struct {
	Composite
}

func (c *Composite) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if strings.EqualFold(a.Name.Local, "operator") {
			c.Operator = a.Value
		}
	}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "fileDependency":
				var v FileDep
				if err := d.DecodeElement(&v, &t); err != nil {
					return err
				}
				c.Files = append(c.Files, v)
			case "flagDependency":
				var v FlagDep
				if err := d.DecodeElement(&v, &t); err != nil {
					return err
				}
				c.Flags = append(c.Flags, v)
			case "gameDependency":
				var v GameDep
				if err := d.DecodeElement(&v, &t); err != nil {
					return err
				}
				c.Games = append(c.Games, v)
			case "dependencies", "visible", "compositeDependency":
				var n Composite
				if err := d.DecodeElement(&n, &t); err != nil {
					return err
				}
				c.Nested = append(c.Nested, n)
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				return nil
			}
		}
	}
}

func (f *FileDep) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch strings.ToLower(a.Name.Local) {
		case "file":
			f.File = a.Value
		case "state":
			f.State = a.Value
		}
	}
	return d.Skip()
}

func (f *FlagDep) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch strings.ToLower(a.Name.Local) {
		case "flag":
			f.Flag = a.Value
		case "value":
			f.Value = a.Value
		}
	}
	return d.Skip()
}

func (g *GameDep) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if strings.EqualFold(a.Name.Local, "version") {
			g.Version = a.Value
		}
	}
	return d.Skip()
}
