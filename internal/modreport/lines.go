package modreport

import "github.com/Rethunk-Tech/mortar/internal/launch"

const maxErrorLines = 20

// ErrorLines lists up to maxErrorLines formatted SMAPI log lines where modName logged ERROR or ALERT.
func ErrorLines(log, modName string) []string {
	return EntryErrorLines(launch.ParseLog(log), modName)
}

// EntryErrorLines is ErrorLines over entries already parsed.
func EntryErrorLines(entries []launch.Entry, modName string) []string {
	if modName == "" {
		return nil
	}
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
			out = append(out, launch.FormatLog(entries[j:j+1]))
			if len(out) >= maxErrorLines {
				return out
			}
		}
	}
	return out
}
