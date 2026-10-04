package modreport

import (
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

const maxErrorLines = 20

const levelWidth = 5

// ErrorLines lists up to maxErrorLines formatted SMAPI log lines where modName logged ERROR or ALERT.
func ErrorLines(log, modName string) []string {
	if modName == "" {
		return nil
	}
	entries := launch.ParseLog(log)
	out := make([]string, 0, maxErrorLines)
	for i := 0; i < len(entries) && len(out) < maxErrorLines; i++ {
		e := entries[i]
		if e.Cont {
			continue
		}
		if e.Level != launch.Error && e.Level != launch.Alert {
			continue
		}
		if e.Mod != modName {
			continue
		}
		for j := i; j < len(entries) && (j == i || entries[j].Cont); j++ {
			out = append(out, formatEntry(entries[j]))
			if len(out) >= maxErrorLines {
				return out
			}
		}
	}
	return out
}

func formatEntry(e launch.Entry) string {
	if e.Cont || e.Time == "" {
		return e.Message
	}
	level := string(e.Level)
	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(e.Time)
	b.WriteByte(' ')
	b.WriteString(level)
	for n := len(level); n < levelWidth; n++ {
		b.WriteByte(' ')
	}
	b.WriteString(" ")
	b.WriteString(e.Mod)
	b.WriteString("] ")
	b.WriteString(e.Message)
	return b.String()
}
