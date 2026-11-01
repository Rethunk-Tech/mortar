package configsvc

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-cfgedit"
	"github.com/Rethunk-Tech/go-dotnet"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// BepInEx writes a plugin's .cfg the first time the game runs it, and rewrites its defaults only then. A plugin's DLL
// says what it binds before that, so a rebuilt or updated mod shows its new defaults and settings, and a mod the game
// has not run yet can be set up, without a launch.

// bound is a plugin's settings as its DLL binds them.
type bound struct {
	guid     string
	dll      profile.PluginDLL
	at       time.Time
	settings []dotnet.Setting
}

type dllRead struct {
	size     int64
	at       time.Time
	settings []dotnet.Setting
}

// dllCache keeps each DLL's settings while its size and time are unchanged, and the profile's plugin list for a moment:
// the config page asks for every mod's files at once.
type dllCache struct {
	mu    sync.Mutex
	reads map[string]dllRead
	lists map[string]dllList
}

type dllList struct {
	at   time.Time
	dlls map[string]profile.PluginDLL
}

// listFor bounds how long the profile's plugin list is reused; a build installed a moment later shows on the next look.
const listFor = 3 * time.Second

func (s *Service) pluginDLLs(game, profileID string) map[string]profile.PluginDLL {
	key := game + "\x00" + profileID
	s.dlls.mu.Lock()
	if l, ok := s.dlls.lists[key]; ok && time.Since(l.at) < listFor {
		s.dlls.mu.Unlock()
		return l.dlls
	}
	s.dlls.mu.Unlock()
	dlls := s.Profiles.PluginDLLs(game, profileID)
	s.dlls.mu.Lock()
	if s.dlls.lists == nil {
		s.dlls.lists = map[string]dllList{}
	}
	s.dlls.lists[key] = dllList{at: time.Now(), dlls: dlls}
	s.dlls.mu.Unlock()
	return dlls
}

// boundFor is what the plugin with guid binds, when the profile holds it and its DLL binds anything Mortar can show.
func (s *Service) boundFor(game, profileID, guid string) (bound, bool) {
	if guid == "" {
		return bound{}, false
	}
	for g, dll := range s.pluginDLLs(game, profileID) {
		if !strings.EqualFold(g, guid) {
			continue
		}
		info, err := os.Stat(dll.Path)
		if err != nil {
			return bound{}, false
		}
		settings := s.settingsOf(dll.Path, info)
		settings = slices.DeleteFunc(slices.Clone(settings), func(st dotnet.Setting) bool { return st.Default == "" && st.Type != "string" })
		return bound{guid: g, dll: dll, at: info.ModTime(), settings: settings}, len(settings) > 0
	}
	return bound{}, false
}

func (s *Service) settingsOf(path string, info os.FileInfo) []dotnet.Setting {
	s.dlls.mu.Lock()
	hit, ok := s.dlls.reads[path]
	s.dlls.mu.Unlock()
	if ok && hit.size == info.Size() && hit.at.Equal(info.ModTime()) {
		return hit.settings
	}
	settings, err := dotnet.Settings(path)
	if err != nil {
		settings = nil
	}
	s.dlls.mu.Lock()
	if s.dlls.reads == nil {
		s.dlls.reads = map[string]dllRead{}
	}
	// ponytail: one small entry per plugin DLL ever read, kept until Mortar restarts.
	s.dlls.reads[path] = dllRead{size: info.Size(), at: info.ModTime(), settings: settings}
	s.dlls.mu.Unlock()
	return settings
}

// unwritten is the .cfg each of guids would have, for the plugins whose DLL binds settings but which have no .cfg yet.
func (s *Service) unwritten(game, profileID string, guids []string, docs []cfgOnDisk) []ConfigFile {
	var out []ConfigFile
	for _, g := range guids {
		name := g + cfgExt
		if g == "" || name != filepath.Base(name) || slices.ContainsFunc(docs, func(d cfgOnDisk) bool { return !d.json && !d.text && d.ownedBy([]string{g}) }) {
			continue
		}
		if b, ok := s.boundFor(game, profileID, g); ok {
			out = append(out, ConfigFile{Name: name, Format: FormatBepInEx, Label: cmp.Or(b.dll.Name, g)})
		}
	}
	return out
}

// netTypes is the .NET type name BepInEx writes for each setting type the DLL reader reports.
var netTypes = map[string]string{
	"bool": "Boolean", "int": "Int32", "uint": "UInt32", "long": "Int64", "ulong": "UInt64", "short": "Int16",
	"ushort": "UInt16", "byte": "Byte", "sbyte": "SByte", "float": "Single", "double": "Double", "string": "String",
	"enum": "Enum",
}

// render is the .cfg BepInEx would write for b on first run, in the layout cfgedit reads.
func (b bound) render() string {
	var out strings.Builder
	fmt.Fprintf(&out, "## Settings file was created by plugin %s v%s\n## Plugin GUID: %s\n", cmp.Or(b.dll.Name, b.guid), cmp.Or(b.dll.Version, "0.0.0"), b.guid)
	var sections []string
	for _, st := range b.settings {
		if !slices.Contains(sections, st.Section) {
			sections = append(sections, st.Section)
		}
	}
	for _, sec := range sections {
		fmt.Fprintf(&out, "\n[%s]\n", sec)
		for _, st := range b.settings {
			if st.Section == sec {
				out.WriteString(entryText(st))
			}
		}
	}
	return out.String()
}

// entryText is one setting as BepInEx writes it, after the blank line before it.
func entryText(st dotnet.Setting) string {
	var out strings.Builder
	out.WriteString("\n")
	for line := range strings.SplitSeq(st.Description, "\n") {
		if st.Description != "" {
			fmt.Fprintf(&out, "## %s\n", strings.TrimRight(line, "\r"))
		}
	}
	fmt.Fprintf(&out, "# Setting type: %s\n# Default value: %s\n", netTypes[st.Type], st.Default)
	switch {
	case len(st.Values) > 0:
		fmt.Fprintf(&out, "# Acceptable values: %s\n", strings.Join(st.Values, ", "))
	case st.Min != "" && st.Max != "":
		fmt.Fprintf(&out, "# Acceptable value range: From %s to %s\n", st.Min, st.Max)
	}
	fmt.Fprintf(&out, "%s = %s\n", st.Key, st.Default)
	return out.String()
}

// setting is the setting b binds under section and key.
func (b bound) setting(section, key string) (dotnet.Setting, bool) {
	i := slices.IndexFunc(b.settings, func(st dotnet.Setting) bool { return st.Section == section && st.Key == key })
	if i < 0 {
		return dotnet.Setting{}, false
	}
	return b.settings[i], true
}

// overlay gives the entries of a .cfg older than its plugin's DLL what the DLL now binds: defaults, descriptions,
// ranges and values, and the settings the file does not have yet, each at its default.
func overlay(sc Schema, b bound) Schema {
	fresh := cfgSchema(cfgedit.ParseBepInEx(b.render()), sc.File)
	for _, fs := range fresh.Sections {
		for _, fe := range fs.Entries {
			i := slices.IndexFunc(sc.Sections, func(s Section) bool { return s.Name == fs.Name })
			if i < 0 {
				sc.Sections = append(sc.Sections, Section{Name: fs.Name, Entries: []Entry{fe}})
				continue
			}
			sec := &sc.Sections[i]
			j := slices.IndexFunc(sec.Entries, func(e Entry) bool { return e.Key == fe.Key })
			if j < 0 {
				sec.Entries = append(sec.Entries, fe)
				continue
			}
			e := &sec.Entries[j]
			e.Default, e.HasDefault = fe.Default, true
			e.Description = cmp.Or(fe.Description, e.Description)
			if fe.Min != nil {
				e.Min, e.Max = fe.Min, fe.Max
			}
			if len(fe.Values) > 0 {
				e.Values = fe.Values
			}
		}
	}
	return sc
}

// syncLevels names the DLL reader's sync levels as ConfigSync's sidecar does.
var syncLevels = map[string]string{"host-enforced": "hostEnforced", "host-recommended": "hostRecommended"}

// boundSync marks the entries b shares with the lobby, for when ConfigSync has not written its sidecar for this build.
func boundSync(sc Schema, b bound) Schema {
	for i := range sc.Sections {
		for j := range sc.Sections[i].Entries {
			e := &sc.Sections[i].Entries[j]
			if st, ok := b.setting(sc.Sections[i].Name, e.Key); ok && st.Sync != "" {
				e.Sync = syncLevels[st.Sync]
			}
		}
	}
	return sc
}

// guidOf is the plugin a .cfg belongs to: the GUID its header names, else its file name.
func guidOf(doc *cfgedit.BepInEx, name string) string {
	return cmp.Or(doc.GUID(), strings.TrimSuffix(name, filepath.Ext(name)))
}

// withBound applies what the plugin's DLL binds to the schema of its .cfg at path (modified at cfgAt): defaults and
// new settings when the DLL is newer than the file, and sync levels when ConfigSync's sidecar is missing or older.
func (s *Service) withBound(game, profileID string, sc Schema, guid, path string, cfgAt time.Time) Schema {
	b, ok := s.boundFor(game, profileID, guid)
	if !ok {
		return sc
	}
	if b.at.After(cfgAt) {
		sc = overlay(sc, b)
	}
	side, err := os.Stat(strings.TrimSuffix(path, filepath.Ext(path)) + syncSidecar)
	if err != nil || b.at.After(side.ModTime()) {
		sc = boundSync(sc, b)
	}
	return sc
}
