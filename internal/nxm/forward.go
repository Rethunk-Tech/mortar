package nxm

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// LinkGame is the nxm URL's game segment (the host), or an error when raw is not an nxm link.
func LinkGame(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "nxm" || u.Host == "" {
		return "", reject(ReasonForm)
	}
	return u.Host, nil
}

// WindowsForwardArgv is argv for the saved Windows handler, with link as one element in place of %1.
func WindowsForwardArgv(previousSaved, link string) (string, []string, error) {
	p := splitPrevious(previousSaved)
	if strings.TrimSpace(p.cmd) == "" {
		return "", nil, errors.New("no previous nxm handler")
	}
	argv := commandLineToArgv(p.cmd)
	if len(argv) == 0 || argv[0] == "" {
		return "", nil, errors.New("no previous nxm handler")
	}
	return programAndArgs(argv, "%1", link)
}

// LinuxForwardArgv is argv for the saved desktop Exec, with link as one element in place of %u or %U.
func LinuxForwardArgv(previousDesktopID, link string, execLine func(string) (string, error)) (string, []string, error) {
	if previousDesktopID == "" {
		return "", nil, errors.New("no previous nxm handler")
	}
	if execLine == nil {
		return "", nil, errors.New("no desktop lookup")
	}
	exec, err := execLine(previousDesktopID)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(exec) == "" {
		return "", nil, fmt.Errorf("desktop entry %q has no Exec", previousDesktopID)
	}
	argv := splitDesktopExec(exec)
	if len(argv) == 0 || argv[0] == "" {
		return "", nil, fmt.Errorf("desktop entry %q has no Exec", previousDesktopID)
	}
	return programAndArgs(argv, "", link)
}

func programAndArgs(argv []string, winField, link string) (string, []string, error) {
	placed := false
	out := make([]string, 0, len(argv)+1)
	for _, a := range argv {
		if winField != "" {
			if before, after, ok := strings.Cut(a, winField); ok {
				out = append(out, before+link+after)
				placed = true
				continue
			}
			out = append(out, a)
			continue
		}
		switch a {
		case "%u", "%U":
			out = append(out, link)
			placed = true
		case "%f", "%F", "%i", "%c", "%k":
		default:
			out = append(out, a)
		}
	}
	if !placed {
		out = append(out, link)
	}
	return out[0], out[1:], nil
}

// commandLineToArgv splits a Windows command line under CommandLineToArgvW rules.
func commandLineToArgv(cmd string) []string {
	var args []string
	var arg strings.Builder
	inQuotes := false
	bs := 0
	started := false
	flush := func() {
		args = append(args, arg.String())
		arg.Reset()
		started = false
	}
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if c == '\\' {
			bs++
			started = true
			continue
		}
		if c == '"' {
			if bs%2 == 0 {
				arg.WriteString(strings.Repeat(`\`, bs/2))
				inQuotes = !inQuotes
			} else {
				arg.WriteString(strings.Repeat(`\`, bs/2))
				arg.WriteByte('"')
			}
			bs = 0
			started = true
			continue
		}
		if bs > 0 {
			arg.WriteString(strings.Repeat(`\`, bs))
			bs = 0
		}
		if (c == ' ' || c == '\t') && !inQuotes {
			if started {
				flush()
			}
			continue
		}
		arg.WriteByte(c)
		started = true
	}
	if bs > 0 {
		arg.WriteString(strings.Repeat(`\`, bs))
		started = true
	}
	if started {
		flush()
	}
	return args
}

func splitDesktopExec(exec string) []string {
	var args []string
	i := 0
	n := len(exec)
	quote := byte('"')
	for i < n {
		for i < n && unicode.IsSpace(rune(exec[i])) {
			i++
		}
		if i >= n {
			break
		}
		if exec[i] == quote {
			arg, next := readQuotedExec(exec, i+1)
			args = append(args, arg)
			i = next
			continue
		}
		start := i
		for i < n && !unicode.IsSpace(rune(exec[i])) && exec[i] != quote {
			i++
		}
		args = append(args, exec[start:i])
	}
	return args
}

func readQuotedExec(exec string, i int) (string, int) {
	var b strings.Builder
	quote := byte('"')
	for i < len(exec) {
		c := exec[i]
		if c == '\\' && i+1 < len(exec) {
			next := exec[i+1]
			if next == quote || next == '`' || next == '$' || next == '\\' {
				b.WriteByte(next)
				i += 2
				continue
			}
		}
		if c == quote {
			return b.String(), i + 1
		}
		b.WriteByte(c)
		i++
	}
	return b.String(), i
}
