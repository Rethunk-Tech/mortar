package nxm

import "strings"

// previous is what the nxm key held before Mortar took it over. Saved as "command\nicon\nname"; a value saved
// with fewer lines restores only the parts it holds.
type previous struct {
	cmd, icon, name  string
	hasIcon, hasName bool
}

func previousID(cmd, icon, name string) string {
	if cmd == "" {
		return ""
	}
	return cmd + "\n" + icon + "\n" + name
}

func splitPrevious(saved string) previous {
	if saved == "" {
		return previous{}
	}
	parts := strings.SplitN(saved, "\n", 3)
	p := previous{cmd: parts[0], hasIcon: len(parts) > 1, hasName: len(parts) > 2}
	if p.hasIcon {
		p.icon = parts[1]
	}
	if p.hasName {
		p.name = parts[2]
	}
	return p
}
