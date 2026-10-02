package fomod

import (
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

// PluginType is the effective type of a plugin under flags and file dependencies.
func PluginType(p Plugin, flags map[string]string, ctx EvalContext) string {
	for _, pat := range p.Patterns {
		if pat.Type.Eval(flags, ctx) {
			return normType(pat.Name)
		}
	}
	if p.DefaultType != "" {
		return normType(p.DefaultType)
	}
	return normType(p.TypeName)
}

func normType(s string) string {
	switch {
	case strings.EqualFold(s, TypeRequired):
		return TypeRequired
	case strings.EqualFold(s, TypeRecommended):
		return TypeRecommended
	case strings.EqualFold(s, TypeNotUsable):
		return TypeNotUsable
	case strings.EqualFold(s, TypeCouldBeUsable):
		return TypeCouldBeUsable
	default:
		return TypeOptional
	}
}

// Eval reports whether the composite dependency holds.
func (c Composite) Eval(flags map[string]string, ctx EvalContext) bool {
	or := strings.EqualFold(c.Operator, "Or")
	n := len(c.Files) + len(c.Flags) + len(c.Games) + len(c.Nested)
	if n == 0 {
		return !or
	}
	ok := func(v bool) bool {
		if or {
			return v
		}
		return !v
	}
	for _, f := range c.Files {
		if ok(fileOK(f, ctx.Files)) {
			return or
		}
	}
	for _, f := range c.Flags {
		if ok(flags[f.Flag] == f.Value) {
			return or
		}
	}
	for _, g := range c.Games {
		if ok(gameDepOK(ctx.GameVersion, g.Version)) {
			return or
		}
	}
	for _, n := range c.Nested {
		if ok(n.Eval(flags, ctx)) {
			return or
		}
	}
	return !or
}

func gameDepOK(installed, minimum string) bool {
	if minimum == "" {
		return true
	}
	if installed == "" {
		return false
	}
	c, ok := meta.CompareVersions(installed, minimum)
	return !ok || c >= 0
}

func fileOK(d FileDep, files FileIndex) bool {
	want := d.State
	if want == "" {
		want = FileActive
	}
	got := FileMissing
	if files != nil {
		got = files(d.File)
	}
	return strings.EqualFold(got, want)
}

// VisibleSteps are install steps whose visible condition holds.
func VisibleSteps(cfg Config, flags map[string]string, ctx EvalContext) []Step {
	var out []Step
	for _, s := range cfg.Steps {
		if s.Visible.Eval(flags, ctx) {
			out = append(out, s)
		}
	}
	return out
}

// FlagsFrom returns condition flags set by the selected plugins, in step order.
func FlagsFrom(cfg Config, choices Choices, ctx EvalContext) map[string]string {
	flags := map[string]string{}
	for _, s := range cfg.Steps {
		if !s.Visible.Eval(flags, ctx) {
			continue
		}
		for _, g := range s.Groups {
			for _, name := range selected(choices, s.Name, g.Name, g) {
				p, ok := plugin(g, name)
				if !ok {
					continue
				}
				for _, f := range p.Flags {
					flags[f.Name] = f.Value
				}
			}
		}
	}
	return flags
}

func plugin(g Group, name string) (Plugin, bool) {
	i := slices.IndexFunc(g.Plugins, func(p Plugin) bool { return p.Name == name })
	if i < 0 {
		return Plugin{}, false
	}
	return g.Plugins[i], true
}

func selected(choices Choices, step, group string, g Group) []string {
	if strings.EqualFold(g.Type, SelectAll) {
		names := make([]string, len(g.Plugins))
		for i, p := range g.Plugins {
			names[i] = p.Name
		}
		return names
	}
	names := slices.Clone(choices[step][group])
	for _, p := range g.Plugins {
		if PluginType(p, nil, EvalContext{}) == TypeRequired && !slices.Contains(names, p.Name) {
			names = append(names, p.Name)
		}
	}
	return names
}

// Match reports whether choices still name plugins that exist and satisfy each visible group's type.
func Match(cfg Config, choices Choices, ctx EvalContext) bool {
	flags := map[string]string{}
	for _, s := range cfg.Steps {
		if !s.Visible.Eval(flags, ctx) {
			continue
		}
		for _, g := range s.Groups {
			got := selected(choices, s.Name, g.Name, g)
			if !groupOK(g, got) {
				return false
			}
			for _, name := range got {
				p, ok := plugin(g, name)
				if !ok {
					return false
				}
				if PluginType(p, flags, ctx) == TypeNotUsable {
					return false
				}
			}
			for _, name := range got {
				p, _ := plugin(g, name)
				for _, f := range p.Flags {
					flags[f.Name] = f.Value
				}
			}
		}
	}
	return true
}

func groupOK(g Group, names []string) bool {
	n := len(names)
	switch {
	case strings.EqualFold(g.Type, SelectExactlyOne):
		return n == 1
	case strings.EqualFold(g.Type, SelectAtMostOne):
		return n <= 1
	case strings.EqualFold(g.Type, SelectAtLeastOne):
		return n >= 1
	case strings.EqualFold(g.Type, SelectAll):
		return true
	default:
		return true
	}
}

// Resolve lists file operations for required files, selected plugins, and matching conditional installs.
func Resolve(cfg Config, choices Choices, ctx EvalContext) []Op {
	var ops []Op
	ops = append(ops, cfg.Required...)
	flags := map[string]string{}
	for _, s := range cfg.Steps {
		if !s.Visible.Eval(flags, ctx) {
			continue
		}
		for _, g := range s.Groups {
			for _, name := range selected(choices, s.Name, g.Name, g) {
				p, ok := plugin(g, name)
				if !ok {
					continue
				}
				ops = append(ops, p.Files...)
				for _, f := range p.Flags {
					flags[f.Name] = f.Value
				}
			}
		}
	}
	for _, c := range cfg.Conditionals {
		if c.When.Eval(flags, ctx) {
			ops = append(ops, c.Files...)
		}
	}
	slices.SortStableFunc(ops, func(a, b Op) int { return a.Priority - b.Priority })
	return ops
}

// Unanswered reports whether cfg shows a group the user picks from that neither old nor choices knew, so replaying
// choices made against old would decide it without asking.
func Unanswered(old, cfg Config, choices Choices, ctx EvalContext) bool {
	seen := map[[2]string]bool{}
	for _, s := range old.Steps {
		for _, g := range s.Groups {
			seen[[2]string{s.Name, g.Name}] = true
		}
	}
	for _, s := range VisibleSteps(cfg, FlagsFrom(cfg, choices, ctx), ctx) {
		for _, g := range s.Groups {
			if strings.EqualFold(g.Type, SelectAll) || seen[[2]string{s.Name, g.Name}] {
				continue
			}
			if _, ok := choices[s.Name][g.Name]; !ok {
				return true
			}
		}
	}
	return false
}

// Keep returns the choices that still name a plugin of cfg.
func Keep(cfg Config, choices Choices) Choices {
	out := Choices{}
	for _, s := range cfg.Steps {
		for _, g := range s.Groups {
			for _, name := range choices[s.Name][g.Name] {
				if _, ok := plugin(g, name); !ok {
					continue
				}
				if out[s.Name] == nil {
					out[s.Name] = map[string][]string{}
				}
				out[s.Name][g.Name] = append(out[s.Name][g.Name], name)
			}
		}
	}
	return out
}
