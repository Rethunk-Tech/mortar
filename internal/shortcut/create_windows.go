//go:build windows

package shortcut

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// create writes a .lnk in the Start menu's Mortar folder through the shell's own shortcut object.
func create(exe, arg, name string) (string, error) {
	appData, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Mortar")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fileName(name)+".lnk")
	if err := writeLink(path, exe, arg); err != nil {
		return "", err
	}
	return path, nil
}

// writeLink drives WScript.Shell's CreateShortcut, which COM needs on one OS thread for the whole call.
func writeLink(path, exe, arg string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		var oleErr *ole.OleError
		// S_FALSE means COM was already initialised on this thread, which is fine.
		if !errors.As(err, &oleErr) || oleErr.Code() != 1 {
			return err
		}
	}
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unknown.Release()
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer shell.Release()
	linkVar, err := oleutil.CallMethod(shell, "CreateShortcut", path)
	if err != nil {
		return err
	}
	link := linkVar.ToIDispatch()
	defer link.Release()
	for prop, value := range map[string]string{"TargetPath": exe, "Arguments": arg, "IconLocation": exe + ",0"} {
		if _, err := oleutil.PutProperty(link, prop, value); err != nil {
			return err
		}
	}
	_, err = oleutil.CallMethod(link, "Save")
	return err
}

// fileName keeps a shortcut's name to characters every file system accepts.
func fileName(name string) string {
	clean := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < ' ' {
			return '-'
		}
		return r
	}, name)
	clean = strings.Trim(clean, " .")
	if clean == "" {
		return "Mortar profile"
	}
	return clean
}
