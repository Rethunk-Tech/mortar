package nxm

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// LinkGame is the nxm URL's game segment (the host), or an error when raw is not an nxm link.
func LinkGame(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "nxm" || u.Host == "" {
		return "", reject(ReasonForm)
	}
	return u.Host, nil
}

// WindowsForwardArgv is argv to run the saved Windows handler with link in place of %1.
func WindowsForwardArgv(previousSaved, link string) (string, []string, error) {
	p := splitPrevious(previousSaved)
	if p.cmd == "" {
		return "", nil, errors.New("no previous nxm handler")
	}
	line := strings.Replace(p.cmd, "%1", link, 1)
	return "cmd", []string{"/c", line}, nil
}

// LinuxForwardArgv runs the saved desktop entry's Exec with %u replaced by link.
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
	exec = strings.ReplaceAll(exec, "%u", link)
	if exec == "" {
		return "", nil, fmt.Errorf("desktop entry %q has no Exec", previousDesktopID)
	}
	return "sh", []string{"-c", exec}, nil
}
