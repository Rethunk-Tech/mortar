//go:build !windows

package main

import (
	"strings"
	"testing"
)

func TestParseMissingLocation(t *testing.T) {
	cases := []struct {
		tool   string
		exit   int
		stdout string
		want   locationChoice
	}{
		{"zenity", 0, "", choiceDefault},
		{"zenity", 1, labelChoose() + "\n", choiceChoose},
		{"zenity", 1, "", choiceQuit},
		{"kdialog", 0, "", choiceDefault},
		{"kdialog", 1, "", choiceChoose},
		{"kdialog", 2, "", choiceQuit},
		{"kdialog", -1, "", choiceQuit},
	}
	for _, c := range cases {
		if got := parseMissingLocation(c.tool, c.exit, c.stdout); got != c.want {
			t.Errorf("%s exit %d %q = %v, want %v", c.tool, c.exit, c.stdout, got, c.want)
		}
	}
}

func TestStartupErrorOffersTheLogFolder(t *testing.T) {
	cases := []struct {
		tool, folder, stdout string
		exit                 int
		opens                bool
	}{
		{"zenity", "/data", labelOpenLog() + "\n", 1, true},
		{"zenity", "/data", "", 0, false},
		{"kdialog", "/data", "", 0, true},
		{"kdialog", "/data", "", 1, false},
	}
	for _, c := range cases {
		args := strings.Join(startupErrorArgs(c.tool, "boom", c.folder), " ")
		if !strings.Contains(args, labelOpenLog()) {
			t.Errorf("%s args %q offer no log folder", c.tool, args)
		}
		if got := opensLogFolder(c.tool, c.exit, c.stdout); got != c.opens {
			t.Errorf("%s exit %d %q opens = %v", c.tool, c.exit, c.stdout, got)
		}
	}
	if args := strings.Join(startupErrorArgs("zenity", "boom", ""), " "); strings.Contains(args, labelOpenLog()) {
		t.Errorf("no folder still offers it: %q", args)
	}
}
