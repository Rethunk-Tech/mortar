//go:build windows

package main

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows"
)

const (
	mbYesNoCancel = 0x3
	mbOK          = 0x0
	mbIconError   = 0x10
	mbTopmost     = 0x40000
	idYes, idNo   = 6, 7
	bifFolders    = 0x1
)

// askMissingLocation uses a plain message box, since Wails has no window yet. Its buttons cannot be renamed, so
// the text names what each one does.
func askMissingLocation(path string) locationChoice {
	text := fmt.Sprintf("Mortar's data folder was not found:\n\n%s\n\nIts drive may be unplugged. Plug it in and choose "+
		"Cancel, then start Mortar again, or choose here:\n\nYes: Use the default location\nNo: Choose the data folder...\nCancel: Quit", path)
	switch w32.MessageBox(0, text, "Mortar", mbYesNoCancel|mbIconError|mbTopmost) {
	case idYes:
		return choiceDefault
	case idNo:
		return choiceChoose
	}
	return choiceQuit
}

func showStartupError(msg string) {
	w32.MessageBox(0, msg, "Mortar", mbOK|mbIconError|mbTopmost)
}

// pickDataFolder shows the shell's folder browser; "" means cancelled.
func pickDataFolder() string {
	title, err := windows.UTF16PtrFromString("Choose the Mortar data folder")
	if err != nil {
		return ""
	}
	pidl := w32.SHBrowseForFolder(&w32.BROWSEINFO{Title: title, Flags: bifFolders})
	if pidl == 0 {
		return ""
	}
	return w32.SHGetPathFromIDList(pidl)
}
