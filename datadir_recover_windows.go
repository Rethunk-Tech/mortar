//go:build windows

package main

import (
	"log"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/i18n"

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
	text := i18n.T("Mortar's data folder was not found:\n\n{path}\n\nIts drive may be unplugged. Plug it in and choose "+
		"Cancel, then start Mortar again, or choose here:\n\nYes: Use the default location\nNo: Choose the data folder…\nCancel: Quit", "path", path)
	switch w32.MessageBox(0, text, "Mortar", mbYesNoCancel|mbIconError|mbTopmost) {
	case idYes:
		return choiceDefault
	case idNo:
		return choiceChoose
	}
	return choiceQuit
}

// showStartupError shows msg and offers to open folder, the one holding the log ("" for none). A message box copies
// its text with Ctrl+C, which the prompt says, so the error can be pasted into a report.
func showStartupError(msg, folder string) {
	if folder == "" {
		w32.MessageBox(0, msg, "Mortar", mbOK|mbIconError|mbTopmost)
		return
	}
	text := msg + "\n\n" + i18n.T("Open the folder that holds the log? (Ctrl+C copies this message.)")
	if w32.MessageBox(0, text, "Mortar", mbYesNo|mbIconError|mbTopmost) == idYes {
		if err := datadir.Open(folder); err != nil {
			log.Printf("open %s: %v", folder, err)
		}
	}
}

// pickDataFolder shows the shell's folder browser; "" means cancelled.
func pickDataFolder() string {
	title, err := windows.UTF16PtrFromString(i18n.T("Choose the Mortar data folder"))
	if err != nil {
		return ""
	}
	pidl := w32.SHBrowseForFolder(&w32.BROWSEINFO{Title: title, Flags: bifFolders})
	if pidl == 0 {
		return ""
	}
	return w32.SHGetPathFromIDList(pidl)
}
