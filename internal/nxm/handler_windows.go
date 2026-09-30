package nxm

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	classKey   = `Software\Classes\nxm`
	commandKey = classKey + `\shell\open\command`
)

// System is the system's registration of the nxm scheme.
type System struct{ exe string }

// New returns the handler for this system, registering exe.
func New(exe string) (*System, error) { return &System{exe: exe}, nil }

func (w *System) command() string { return `"` + w.exe + `" "%1"` }

func (w *System) Owner() (Owner, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, commandKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return Owner{}, nil
	}
	if err != nil {
		return Owner{}, err
	}
	defer k.Close()
	cmd, _, err := k.GetStringValue("")
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return Owner{}, err
	}
	return Owner{ID: cmd, Name: exeOf(cmd), Mine: cmd == w.command()}, nil
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
	k, _, err := registry.CreateKey(registry.CURRENT_USER, classKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetStringValue("", "URL:NXM Protocol"); err != nil {
		return err
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	return setCommand(w.command())
}

func setCommand(cmd string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, commandKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue("", cmd)
}

func (w *System) Restore(previous string) error {
	if previous != "" {
		return setCommand(previous)
	}
	for _, key := range []string{commandKey, classKey + `\shell\open`, classKey + `\shell`, classKey} {
		if err := registry.DeleteKey(registry.CURRENT_USER, key); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("delete %s: %w", key, err)
		}
	}
	return nil
}
