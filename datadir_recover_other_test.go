//go:build !windows

package main

import "testing"

func TestParseMissingLocation(t *testing.T) {
	cases := []struct {
		tool   string
		exit   int
		stdout string
		want   locationChoice
	}{
		{"zenity", 0, "", choiceDefault},
		{"zenity", 1, labelChoose + "\n", choiceChoose},
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
