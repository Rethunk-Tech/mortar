//go:build windows

package main

import (
	"errors"
	"os"

	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/wailsapp/wails/v3/pkg/w32"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	mbYesNo          = 0x4
	webView2Download = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
	webView2Client   = `Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
)

// webView2Installed is the check Microsoft's WebView2 distribution guide gives: a pv value above 0.0.0.0 under the
// runtime's per-machine or per-user EdgeUpdate client key. A runtime folder named in the environment counts too.
func webView2Installed() bool {
	if os.Getenv("WEBVIEW2_BROWSER_EXECUTABLE_FOLDER") != "" {
		return true
	}
	keys := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\` + webView2Client},
		{registry.LOCAL_MACHINE, `SOFTWARE\` + webView2Client},
		{registry.CURRENT_USER, `Software\` + webView2Client},
	}
	for _, k := range keys {
		h, err := registry.OpenKey(k.root, k.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		pv, _, err := h.GetStringValue("pv")
		_ = h.Close()
		if err == nil && pv != "" && pv != "0.0.0.0" {
			return true
		}
	}
	return false
}

// requireWebView2 asks before Wails starts, since Wails without the runtime opens no window and keeps running, so
// every later launch hands off to that invisible process. Start anyway stays for a runtime the registry misses.
func requireWebView2() error {
	if webView2Installed() {
		return nil
	}
	text := i18n.T("Mortar needs the {runtime}, which does not seem to be installed.\n\n"+
		"Yes: Open the download page (install the {installer}, then start Mortar again)\nNo: Start anyway",
		"runtime", "Microsoft Edge WebView2 Runtime", "installer", "Evergreen Bootstrapper")
	if w32.MessageBox(0, text, "Mortar", mbYesNo|mbIconError|mbTopmost) != idYes {
		return nil
	}
	if checked, err := opener.Web(webView2Download); err == nil {
		if url, err := windows.UTF16PtrFromString(checked); err == nil {
			_ = windows.ShellExecute(0, nil, url, nil, nil, windows.SW_SHOWNORMAL)
		}
	}
	return shownError{errors.New("the WebView2 Runtime is not installed")}
}
