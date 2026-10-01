package tools

import "strings"

// Context holds placeholder values for a profile launch.
type Context struct {
	Game    string
	Mods    string
	Saves   string
	Profile string
}

func expand(s string, ctx Context) string {
	if s == "" {
		return s
	}
	r := strings.NewReplacer(
		"{game}", ctx.Game,
		"{mods}", ctx.Mods,
		"{saves}", ctx.Saves,
		"{profile}", ctx.Profile,
	)
	return r.Replace(s)
}

func expandArgs(args []string, ctx Context) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = expand(a, ctx)
	}
	return out
}
