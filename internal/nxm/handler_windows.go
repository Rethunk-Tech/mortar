package nxm

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	classKey   = `Software\Classes\nxm`
	commandKey = classKey + `\shell\open\command`
)

func defaultIcon(exe string) string { return exe + ",0" }

// System is the system's registration of the nxm scheme. software is the HKCU key the browsers' native messaging
// keys live under.
type System struct{ exe, software string }

// New returns the handler for this system, registering exe.
func New(exe string) (*System, error) { return &System{exe: exe, software: "Software"}, nil }

func (w *System) command() string { return `"` + w.exe + `" "%1"` }

func (w *System) Owner() (Owner, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, commandKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return Owner{}, nil
	}
	if err != nil {
		return Owner{}, err
	}
	defer func() { _ = k.Close() }()
	cmd, _, err := k.GetStringValue("")
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return Owner{}, err
	}
	if cmd == "" {
		return Owner{}, nil
	}
	return Owner{ID: previousID(cmd, readIcon(), readName()), Name: exeOf(cmd), Mine: cmd == w.command()}, nil
}

// exeOf is the program of an open command, for showing to the user.
func exeOf(cmd string) string {
	if rest, ok := strings.CutPrefix(cmd, `"`); ok {
		exe, _, _ := strings.Cut(rest, `"`)
		return exe
	}
	exe, _, _ := strings.Cut(cmd, " ")
	return exe
}

func (w *System) Register() error {
	if err := w.WriteNativeHosts(); err != nil {
		return err
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, classKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	if err := k.SetStringValue("", "URL:NXM Protocol"); err != nil {
		return err
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	icon, _, err := registry.CreateKey(registry.CURRENT_USER, classKey+`\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = icon.Close() }()
	if err := icon.SetStringValue("", defaultIcon(w.exe)); err != nil {
		return err
	}
	return setCommand(w.command())
}

func setCommand(cmd string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, commandKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", cmd)
}

func readIcon() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, classKey+`\DefaultIcon`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue("")
	if err != nil {
		return ""
	}
	return v
}

func setIcon(icon string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, classKey+`\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", icon)
}

func (w *System) Restore(previous string) error {
	if err := w.removeNativeHosts(); err != nil {
		return err
	}
	if previous == "" {
		for _, key := range []string{classKey + `\DefaultIcon`, commandKey, classKey + `\shell\open`, classKey + `\shell`, classKey} {
			if err := registry.DeleteKey(registry.CURRENT_USER, key); err != nil && !errors.Is(err, registry.ErrNotExist) {
				return fmt.Errorf("delete %s: %w", key, err)
			}
		}
		return nil
	}
	p := splitPrevious(previous)
	if err := setCommand(p.cmd); err != nil {
		return err
	}
	if p.hasName {
		if err := setName(p.name); err != nil {
			return err
		}
	}
	if !p.hasIcon {
		return nil
	}
	if p.icon == "" {
		if err := registry.DeleteKey(registry.CURRENT_USER, classKey+`\DefaultIcon`); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	return setIcon(p.icon)
}

// readName returns the nxm key's own default value, the protocol's display name.
func readName() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, classKey, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue("")
	if err != nil {
		return ""
	}
	return v
}

func setName(name string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, classKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", name)
}

// RegisterLinks does nothing: the installer registers the mortar scheme and the .mortar file type.
func (w *System) RegisterLinks() error { return nil }

// Refresh does nothing: the installer fixes where Mortar lives.
func (w *System) Refresh() error { return nil }

// ForwardOther runs the saved open command with link in place of %1.
func (w *System) ForwardOther(link, previous string) error {
	name, args, err := WindowsForwardArgv(previous, link)
	if err != nil {
		return err
	}
	out, err := exec.CommandContext(context.Background(), name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
