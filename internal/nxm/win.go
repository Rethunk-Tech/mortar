package nxm

import "strings"

const (
	classKey   = `Software\Classes\nxm`
	commandKey = classKey + `\shell\open\command`
)

func previousID(cmd, icon string) string {
	if cmd == "" {
		return ""
	}
	return cmd + "\n" + icon
}

func splitPrevious(previous string) (cmd, icon string, restoreIcon bool) {
	if previous == "" {
		return "", "", false
	}
	cmd, icon, restoreIcon = strings.Cut(previous, "\n")
	return cmd, icon, restoreIcon
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
	cmd, icon, restoreIcon := splitPrevious(previous)
	m.set(commandKey, "", cmd)
	if !restoreIcon {
		return
	}
	if icon == "" {
		delete(m, classKey+`\DefaultIcon`)
		return
	}
	m.set(classKey+`\DefaultIcon`, "", icon)
}
