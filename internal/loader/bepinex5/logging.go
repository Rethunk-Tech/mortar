package bepinex5

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

const (
	settingConsole  = "console"
	settingLogLevel = "logLevel"
)

// logLevels maps a log level choice to BepInEx's LogLevels flags, which Console and Disk logging share.
var logLevels = []struct{ choice, flags string }{
	{"quiet", "Fatal, Error, Warning"},
	{"default", "Fatal, Error, Warning, Message, Info"},
	{"debug", "Fatal, Error, Warning, Message, Info, Debug"},
	{"all", "All"},
}

func configFile(profileDir string) string {
	return filepath.Join(profileDir, "BepInEx", "config", "BepInEx.cfg")
}

// LaunchSettings are the profile's BepInEx logging options: the console window from Mortar's marker, since BepInEx's
// own default shows it, and the log level as BepInEx.cfg holds it, BepInEx's default when the file is not written yet.
func (Loader) LaunchSettings(profileDir string) ([]loader.LaunchSetting, error) {
	text, err := readConfig(profileDir)
	if err != nil {
		return nil, err
	}
	levels := cfgGet(text, "Logging.Console", "LogLevels")
	level := "default"
	if levels != "" {
		level = ""
		for _, l := range logLevels {
			if l.flags == levels {
				level = l.choice
			}
		}
	}
	choices := make([]string, len(logLevels))
	for i, l := range logLevels {
		choices[i] = l.choice
	}
	return []loader.LaunchSetting{
		{ID: settingConsole, Value: boolText(readMarker(profileDir).Console)},
		{ID: settingLogLevel, Choices: choices, Value: level},
	}, nil
}

// SetLaunchSetting writes one option into the profile's BepInEx.cfg, creating the file when BepInEx has not.
func (Loader) SetLaunchSetting(profileDir, id, value string) error {
	text, err := readConfig(profileDir)
	if err != nil {
		return err
	}
	switch id {
	case settingConsole:
		if value != "true" && value != "false" {
			return fmt.Errorf("console must be true or false, not %q", value)
		}
		m := readMarker(profileDir)
		m.Console = value == "true"
		if err := writeMarker(profileDir, m); err != nil {
			return err
		}
		text = cfgSet(text, "Logging.Console", "Enabled", value)
	case settingLogLevel:
		i := slices.IndexFunc(logLevels, func(l struct{ choice, flags string }) bool { return l.choice == value })
		if i < 0 {
			return fmt.Errorf("unknown log level %q", value)
		}
		text = cfgSet(text, "Logging.Console", "LogLevels", logLevels[i].flags)
		text = cfgSet(text, "Logging.Disk", "LogLevels", logLevels[i].flags)
	default:
		return fmt.Errorf("BepInEx has no setting %q", id)
	}
	return writeConfig(profileDir, text)
}

// applyConsole sets BepInEx.cfg's console window switch, leaving the file alone when it already says so.
func applyConsole(profileDir string, show bool) error {
	text, err := readConfig(profileDir)
	if err != nil {
		return err
	}
	if cfgGet(text, "Logging.Console", "Enabled") == boolText(show) {
		return nil
	}
	return writeConfig(profileDir, cfgSet(text, "Logging.Console", "Enabled", boolText(show)))
}

func writeConfig(profileDir, text string) error {
	path := configFile(profileDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return fsx.WriteFile(path, []byte(text), 0o600)
}

func readConfig(profileDir string) (string, error) {
	b, err := fsx.ReadFile(configFile(profileDir))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

func boolText(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// sectionRange is the line span of [section]'s body in lines, and whether the section exists.
func sectionRange(lines []string, section string) (start, end int, ok bool) {
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if !ok {
			if t == "["+section+"]" {
				start, ok = i+1, true
			}
			continue
		}
		if strings.HasPrefix(t, "[") {
			return start, i, true
		}
	}
	return start, len(lines), ok
}

func keyOf(line string) (key, val string, ok bool) {
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "#") {
		return "", "", false
	}
	k, v, ok := strings.Cut(t, "=")
	return strings.TrimSpace(k), strings.TrimSpace(v), ok
}

func cfgGet(text, section, key string) string {
	lines := strings.Split(text, "\n")
	start, end, ok := sectionRange(lines, section)
	if !ok {
		return ""
	}
	for _, l := range lines[start:end] {
		if k, v, ok := keyOf(l); ok && k == key {
			return v
		}
	}
	return ""
}

// cfgSet changes only key's line, or adds the key (and the section) when it is missing, leaving every other line and
// the file's line endings as they were.
func cfgSet(text, section, key, value string) string {
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	start, end, ok := sectionRange(lines, section)
	entry := key + " = " + value
	switch {
	case !ok:
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]", "", entry, "")
	default:
		for i := start; i < end; i++ {
			if k, _, ok := keyOf(lines[i]); ok && k == key {
				lines[i] = entry
				return strings.Join(lines, eol)
			}
		}
		lines = slices.Insert(lines, start, entry)
	}
	return strings.Join(lines, eol)
}
