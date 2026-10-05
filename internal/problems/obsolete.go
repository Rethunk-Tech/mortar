package problems

// ObsoleteText reports whether a mod's name or summary says its author marked it obsolete or deprecated, by the rule
// Problems applies to installed mods; the browser extension's mortarObsoleteText mirrors it.
func ObsoleteText(name, summary string) bool {
	return statusMatch(name, true) != "" || statusMatch(summary, false) != ""
}
