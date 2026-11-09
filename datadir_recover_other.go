//go:build !windows

package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/godbus/dbus/v5"
)

func labelDefault() string { return i18n.T("Use the default location") }
func labelChoose() string  { return i18n.T("Choose the data folder…") }
func labelQuit() string    { return i18n.T("Quit") }
func labelOpenLog() string { return i18n.T("Open the log folder") }

// Wails has no window yet, so the pre-window dialogs go through zenity or kdialog when one is installed. Without
// either, the error reaches the terminal and startup.log and the next start can fix data-location by hand.
var dialogTools = []string{"zenity", "kdialog"}

func findDialogTool() string {
	// A server build's window is a browser tab, so a dialog would land on whatever desktop happens to run it.
	if serverBuild {
		return ""
	}
	for _, tool := range dialogTools {
		if _, err := exec.LookPath(tool); err == nil {
			return tool
		}
	}
	return ""
}

func missingLocationText(path string) string {
	return i18n.T("Mortar's data folder was not found:\n\n{path}\n\nIts drive may be unplugged. Plug it in and choose "+
		"{quit}, then start Mortar again, or choose here.", "path", path, "quit", labelQuit())
}

// missingLocationArgs builds the three-way question; zenity reports the extra button on stdout, kdialog by exit code.
func missingLocationArgs(tool, path string) []string {
	text := missingLocationText(path)
	if tool == "kdialog" {
		return []string{
			"--title", "Mortar", "--yes-label", labelDefault(), "--no-label", labelChoose(),
			"--cancel-label", labelQuit(), "--yesnocancel", text,
		}
	}
	return []string{
		"--question", "--title", "Mortar", "--text", text, "--ok-label", labelDefault(),
		"--cancel-label", labelQuit(), "--extra-button", labelChoose(),
	}
}

func parseMissingLocation(tool string, exit int, stdout string) locationChoice {
	if tool == "kdialog" {
		switch exit {
		case 0:
			return choiceDefault
		case 1:
			return choiceChoose
		}
		return choiceQuit
	}
	switch {
	case exit == 0:
		return choiceDefault
	case strings.TrimSpace(stdout) == labelChoose():
		return choiceChoose
	}
	return choiceQuit
}

func askMissingLocation(path string) locationChoice {
	tool := findDialogTool()
	if tool == "" {
		return choiceNoDialog
	}
	exit, out := runDialog(tool, missingLocationArgs(tool, path))
	return parseMissingLocation(tool, exit, out)
}

// showStartupError shows msg with a button that opens folder, the one holding the log ("" for none). Without a dialog
// tool it becomes a desktop notification, which every desktop with a notification daemon shows.
func showStartupError(msg, folder string) {
	if serverBuild {
		slog.Error("startup", "msg", msg)
		return
	}
	tool := findDialogTool()
	if tool == "" {
		notifyStartupError(msg)
		return
	}
	if exit, out := runDialog(tool, startupErrorArgs(tool, msg, folder)); folder != "" && opensLogFolder(tool, exit, out) {
		if err := datadir.Open(folder); err != nil {
			slog.Warn("open the log folder", "folder", folder, "err", err)
		}
	}
}

func startupErrorArgs(tool, msg, folder string) []string {
	switch {
	case tool == "kdialog" && folder != "":
		return []string{"--title", "Mortar", "--yes-label", labelOpenLog(), "--no-label", i18n.T("Close"), "--warningyesno", msg}
	case tool == "kdialog":
		return []string{"--title", "Mortar", "--error", msg}
	case folder != "":
		return []string{"--error", "--title", "Mortar", "--text", msg, "--extra-button", labelOpenLog()}
	}
	return []string{"--error", "--title", "Mortar", "--text", msg}
}

// opensLogFolder reads the answer: zenity prints an extra button's label, kdialog's Yes exits 0.
func opensLogFolder(tool string, exit int, stdout string) bool {
	if tool == "kdialog" {
		return exit == 0
	}
	return strings.TrimSpace(stdout) == labelOpenLog()
}

// notifyStartupError prefers the Notification portal, the only route a Flatpak build has without a talk-name on the
// notification service, and falls back to the service itself on a desktop without the portal.
func notifyStartupError(msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := portal.AddNotification(ctx, portal.Notification{ID: "startup-error", Title: "Mortar", Body: msg, Priority: portal.PriorityUrgent})
	if err == nil {
		return
	}
	slog.Warn("startup error notification through the portal", "err", err)
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		slog.Warn("startup error notification", "err", err)
		return
	}
	defer func() { _ = conn.Close() }()
	urgent := map[string]dbus.Variant{"urgency": dbus.MakeVariant(byte(2))}
	call := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications").CallWithContext(ctx,
		"org.freedesktop.Notifications.Notify", 0, "Mortar", uint32(0), "dialog-error", "Mortar", msg, []string{}, urgent, int32(0))
	if call.Err != nil {
		slog.Warn("startup error notification", "err", call.Err)
	}
}

// pickDataFolder shows a folder chooser; "" means cancelled or no dialog tool.
func pickDataFolder() string {
	tool := findDialogTool()
	var args []string
	switch tool {
	case "":
		return ""
	case "kdialog":
		args = []string{"--title", i18n.T("Choose the Mortar data folder"), "--getexistingdirectory"}
	default:
		args = []string{"--file-selection", "--directory", "--title", i18n.T("Choose the Mortar data folder")}
	}
	if exit, out := runDialog(tool, args); exit == 0 {
		return strings.TrimSpace(out)
	}
	return ""
}

// runDialog returns the exit code (-1 when the tool could not run) and stdout.
func runDialog(tool string, args []string) (int, string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, tool, args...) //nolint:gosec // G204: tool is zenity or kdialog, chosen from a fixed pair
	cmd.Stdout = &out
	err := cmd.Run()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode(), out.String()
	}
	if err != nil {
		return -1, ""
	}
	return 0, out.String()
}
