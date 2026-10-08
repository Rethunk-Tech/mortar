package nxm

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/nowindow"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func classKey(scheme string) string   { return `Software\Classes\` + scheme }
func commandKey(scheme string) string { return classKey(scheme) + `\shell\open\command` }

func defaultIcon(exe string) string { return exe + ",0" }

// System is the system's registration of the source link schemes. software is the HKCU key the browsers' native messaging
// keys live under.
type System struct{ exe, software string }

// New returns the handler for this system, registering exe.
func New(exe string) (*System, error) { return &System{exe: exe, software: "Software"}, nil }

func (w *System) command() string { return `"` + w.exe + `" "%1"` }

func (w *System) Owner(scheme string) (Owner, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, commandKey(scheme), registry.QUERY_VALUE)
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
	return Owner{ID: previousID(cmd, readIcon(scheme), readName(scheme)), Name: exeOf(cmd), Mine: sameExe(exeOf(cmd), w.exe)}, nil
}

// sameExe compares program paths the way Windows does: case-insensitively, with 8.3 short names expanded.
func sameExe(a, b string) bool {
	return strings.EqualFold(longPath(a), longPath(b))
}

func longPath(p string) string {
	in, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return p
	}
	n, err := windows.GetLongPathName(in, nil, 0)
	if err != nil || n == 0 {
		return p
	}
	buf := make([]uint16, n)
	n, err = windows.GetLongPathName(in, &buf[0], n)
	if err != nil || n == 0 || int(n) > len(buf) {
		return p
	}
	return windows.UTF16ToString(buf[:n])
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
	for _, scheme := range source.Schemes() {
		if err := w.registerScheme(scheme); err != nil {
			return err
		}
	}
	return nil
}

func (w *System) RegisterSchemes(schemes []string) error {
	for _, scheme := range schemes {
		if err := w.registerScheme(scheme); err != nil {
			return err
		}
	}
	return nil
}

func (w *System) registerScheme(scheme string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, classKey(scheme), registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	if err := k.SetStringValue("", "URL:"+strings.ToUpper(scheme)+" Protocol"); err != nil {
		return err
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	icon, _, err := registry.CreateKey(registry.CURRENT_USER, classKey(scheme)+`\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = icon.Close() }()
	if err := icon.SetStringValue("", defaultIcon(w.exe)); err != nil {
		return err
	}
	return setCommand(scheme, w.command())
}

func setCommand(scheme, cmd string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, commandKey(scheme), registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", cmd)
}

func readIcon(scheme string) string {
	k, err := registry.OpenKey(registry.CURRENT_USER, classKey(scheme)+`\DefaultIcon`, registry.QUERY_VALUE)
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

func setIcon(scheme, icon string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, classKey(scheme)+`\DefaultIcon`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", icon)
}

// Restore hands each claimed scheme back to its previous owner; a scheme with none has its keys deleted. A scheme another
// app owns is left alone. Every scheme is tried and the errors are joined.
func (w *System) Restore(previous map[string]string) error {
	errs := []error{w.removeNativeHosts()}
	for _, scheme := range source.Schemes() {
		errs = append(errs, w.restoreScheme(scheme, previous[scheme]))
	}
	return errors.Join(errs...)
}

// Release hands the given schemes back to their previous owners.
func (w *System) Release(schemes []string, previous map[string]string) error {
	var errs []error
	for _, scheme := range schemes {
		errs = append(errs, w.restoreScheme(scheme, previous[scheme]))
	}
	return errors.Join(errs...)
}

// restoreScheme acts only on a scheme Mortar holds: another app's registration is not Mortar's to delete or replace.
func (w *System) restoreScheme(scheme, previous string) error {
	if o, err := w.Owner(scheme); err != nil || !o.Mine {
		return err
	}
	if previous == "" {
		class := classKey(scheme)
		for _, key := range []string{class + `\DefaultIcon`, commandKey(scheme), class + `\shell\open`, class + `\shell`, class} {
			if err := registry.DeleteKey(registry.CURRENT_USER, key); err != nil && !errors.Is(err, registry.ErrNotExist) {
				return fmt.Errorf("delete %s: %w", key, err)
			}
		}
		return nil
	}
	p := splitPrevious(previous)
	if err := setCommand(scheme, p.cmd); err != nil {
		return err
	}
	if p.hasName {
		if err := setName(scheme, p.name); err != nil {
			return err
		}
	}
	if !p.hasIcon {
		return nil
	}
	if p.icon == "" {
		if err := registry.DeleteKey(registry.CURRENT_USER, classKey(scheme)+`\DefaultIcon`); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	return setIcon(scheme, p.icon)
}

// readName returns the scheme key's own default value, the protocol's display name.
func readName(scheme string) string {
	k, err := registry.OpenKey(registry.CURRENT_USER, classKey(scheme), registry.QUERY_VALUE)
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

func setName(scheme, name string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, classKey(scheme), registry.SET_VALUE)
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
	cmd := exec.CommandContext(context.Background(), name, args...)
	nowindow.Set(cmd)
	// The previous handler may run until the user closes it; waiting would stall every later handoff.
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
