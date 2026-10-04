//go:build !windows

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	labelDefault = "Use the default location"
	labelChoose  = "Choose the data folder…"
	labelQuit    = "Quit"
)

// Wails has no window yet, so the pre-window dialogs go through zenity or kdialog when one is installed. Without
// either, the error reaches the terminal and startup.log and the next start can fix data-location by hand.
var dialogTools = []string{"zenity", "kdialog"}

func findDialogTool() string {
	for _, tool := range dialogTools {
		if _, err := exec.LookPath(tool); err == nil {
			return tool
		}
	}
	return ""
}

func missingLocationText(path string) string {
	return fmt.Sprintf("Mortar's data folder was not found:\n\n%s\n\nIts drive may be unplugged. Plug it in and choose "+
		"Quit, then start Mortar again, or choose here.", path)
}

// missingLocationArgs builds the three-way question; zenity reports the extra button on stdout, kdialog by exit code.
func missingLocationArgs(tool, path string) []string {
	text := missingLocationText(path)
	if tool == "kdialog" {
		return []string{
			"--title", "Mortar", "--yes-label", labelDefault, "--no-label", labelChoose,
			"--cancel-label", labelQuit, "--yesnocancel", text,
		}
	}
	return []string{
		"--question", "--title", "Mortar", "--text", text, "--ok-label", labelDefault,
		"--cancel-label", labelQuit, "--extra-button", labelChoose,
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
	case strings.TrimSpace(stdout) == labelChoose:
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

func showStartupError(msg string) {
	tool := findDialogTool()
	switch tool {
	case "":
	case "kdialog":
		runDialog(tool, []string{"--title", "Mortar", "--error", msg})
	default:
		runDialog(tool, []string{"--error", "--title", "Mortar", "--text", msg})
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
		args = []string{"--title", "Choose the Mortar data folder", "--getexistingdirectory"}
	default:
		args = []string{"--file-selection", "--directory", "--title", "Choose the Mortar data folder"}
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
	cmd := exec.CommandContext(ctx, tool, args...)
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
