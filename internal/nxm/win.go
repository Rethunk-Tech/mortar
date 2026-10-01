package nxm

import "strings"

const (
	classKey   = `Software\Classes\nxm`
	commandKey = classKey + `\shell\open\command`
)

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

// memReg is an injectable HKCU\Software\Classes tree for tests (path → value name → data).
type memReg map[string]map[string]string

func (m memReg) set(path, name, value string) {
	k, ok := m[path]
	if !ok {
		k = map[string]string{}
		m[path] = k
	}
	k[name] = value
}

func (m memReg) register(exe string) {
	m.set(classKey, "", "URL:NXM Protocol")
	m.set(classKey, "URL Protocol", "")
	m.set(classKey+`\DefaultIcon`, "", defaultIcon(exe))
	m.set(commandKey, "", `"`+exe+`" "%1"`)
}

func (m memReg) restore(previous string) {
	if previous == "" {
		for _, key := range []string{classKey + `\DefaultIcon`, commandKey, classKey + `\shell\open`, classKey + `\shell`, classKey} {
			delete(m, key)
		}
		return
	}
	p := splitPrevious(previous)
	m.set(commandKey, "", p.cmd)
	if p.hasName {
		m.set(classKey, "", p.name)
	}
	if !p.hasIcon {
		return
	}
	if p.icon == "" {
		delete(m, classKey+`\DefaultIcon`)
		return
	}
	m.set(classKey+`\DefaultIcon`, "", p.icon)
}
