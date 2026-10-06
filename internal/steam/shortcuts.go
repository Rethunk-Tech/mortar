package steam

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Binary VDF node types, as Steam writes shortcuts.vdf.
const (
	vdfMap    byte = 0x00
	vdfString byte = 0x01
	vdfInt32  byte = 0x02
	vdfUint64 byte = 0x07
	vdfEnd    byte = 0x08
)

// vdfNode is one key of a binary VDF map, kept in file order so a rewrite changes nothing it does not mean to.
type vdfNode struct {
	Kind  byte
	Key   string
	Str   string
	Int   uint32
	Big   uint64
	Child []vdfNode
}

func readCString(r *bufio.Reader) (string, error) {
	s, err := r.ReadString(0)
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], nil
}

func readVDFMap(r *bufio.Reader) ([]vdfNode, error) {
	var out []vdfNode
	for {
		kind, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if kind == vdfEnd {
			return out, nil
		}
		key, err := readCString(r)
		if err != nil {
			return nil, err
		}
		n := vdfNode{Kind: kind, Key: key}
		switch kind {
		case vdfMap:
			if n.Child, err = readVDFMap(r); err != nil {
				return nil, err
			}
		case vdfString:
			if n.Str, err = readCString(r); err != nil {
				return nil, err
			}
		case vdfInt32:
			err = binary.Read(r, binary.LittleEndian, &n.Int)
		case vdfUint64:
			err = binary.Read(r, binary.LittleEndian, &n.Big)
		default:
			return nil, fmt.Errorf("unknown binary VDF node type %#x", kind)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
}

func writeVDFMap(w *bytes.Buffer, nodes []vdfNode) {
	for _, n := range nodes {
		w.WriteByte(n.Kind)
		w.WriteString(n.Key)
		w.WriteByte(0)
		switch n.Kind {
		case vdfMap:
			writeVDFMap(w, n.Child)
		case vdfString:
			w.WriteString(n.Str)
			w.WriteByte(0)
		case vdfInt32:
			_ = binary.Write(w, binary.LittleEndian, n.Int)
		case vdfUint64:
			_ = binary.Write(w, binary.LittleEndian, n.Big)
		}
	}
	w.WriteByte(vdfEnd)
}

// Shortcut is a non-Steam game entry in the Steam library.
type Shortcut struct {
	Name          string
	Exe           string
	StartDir      string
	LaunchOptions string
	// Key is the launch option that names what the entry plays; an entry for the same exe whose options include it
	// is this shortcut, whatever its other options and name, and is updated in place. Empty means the whole
	// LaunchOptions.
	Key   string
	Cover string
}

func (sc Shortcut) is(e vdfNode) bool {
	if field(e, "Exe") != quoted(sc.Exe) {
		return false
	}
	if sc.Key == "" {
		return field(e, "LaunchOptions") == sc.LaunchOptions
	}
	return slices.Contains(strings.Fields(field(e, "LaunchOptions")), sc.Key)
}

// update sets e's name, folder and launch options to sc's, keeping the rest (appid, playtime, the user's tags), and
// reports whether anything changed.
func (sc Shortcut) update(e *vdfNode) bool {
	want := map[string]string{"AppName": sc.Name, "StartDir": quoted(sc.StartDir), "LaunchOptions": sc.LaunchOptions}
	changed := false
	for i, c := range e.Child {
		if v, ok := want[c.Key]; ok && c.Kind == vdfString && c.Str != v {
			e.Child[i].Str = v
			changed = true
		}
	}
	return changed
}

func appIDOf(e vdfNode) uint32 {
	for _, c := range e.Child {
		if c.Key == "appid" && c.Kind == vdfInt32 {
			return c.Int
		}
	}
	return shortcutAppID(field(e, "Exe"), field(e, "AppName"))
}

// shortcutAppID is the id Steam derives for a non-Steam shortcut, from its quoted exe and name.
func shortcutAppID(exe, name string) uint32 {
	return crc32.ChecksumIEEE([]byte(exe+name)) | 0x80000000
}

func quoted(s string) string { return `"` + s + `"` }

func str(key, value string) vdfNode { return vdfNode{Kind: vdfString, Key: key, Str: value} }

func i32(key string, value uint32) vdfNode { return vdfNode{Kind: vdfInt32, Key: key, Int: value} }

func (sc Shortcut) node(index int) vdfNode {
	exe := quoted(sc.Exe)
	return vdfNode{Kind: vdfMap, Key: strconv.Itoa(index), Child: []vdfNode{
		i32("appid", shortcutAppID(exe, sc.Name)),
		str("AppName", sc.Name),
		str("Exe", exe),
		str("StartDir", quoted(sc.StartDir)),
		str("icon", ""),
		str("ShortcutPath", ""),
		str("LaunchOptions", sc.LaunchOptions),
		i32("IsHidden", 0),
		i32("AllowDesktopConfig", 1),
		i32("AllowOverlay", 1),
		i32("OpenVR", 0),
		i32("Devkit", 0),
		str("DevkitGameID", ""),
		i32("DevkitOverrideAppID", 0),
		i32("LastPlayTime", 0),
		str("FlatpakAppID", ""),
		{Kind: vdfMap, Key: "tags"},
	}}
}

func field(n vdfNode, key string) string {
	for _, c := range n.Child {
		if c.Key == key && c.Kind == vdfString {
			return c.Str
		}
	}
	return ""
}

// addShortcut appends sc to a shortcuts.vdf body (empty for a new file), or updates the entry that already is sc in
// place. It returns the entry's appid, which an updated entry keeps so Steam keeps its playtime, and whether the body
// changed.
func addShortcut(body []byte, sc Shortcut) ([]byte, uint32, bool, error) {
	root := []vdfNode{{Kind: vdfMap, Key: "shortcuts"}}
	if len(body) > 0 {
		var err error
		if root, err = readVDFMap(bufio.NewReader(bytes.NewReader(body))); err != nil {
			return nil, 0, false, fmt.Errorf("read shortcuts.vdf: %w", err)
		}
	}
	if len(root) != 1 || root[0].Kind != vdfMap || root[0].Key != "shortcuts" {
		return nil, 0, false, errors.New("shortcuts.vdf has no shortcuts list")
	}
	i := slices.IndexFunc(root[0].Child, sc.is)
	if i >= 0 && !sc.update(&root[0].Child[i]) {
		return body, appIDOf(root[0].Child[i]), false, nil
	}
	if i < 0 {
		i = len(root[0].Child)
		root[0].Child = append(root[0].Child, sc.node(i))
	}
	var buf bytes.Buffer
	writeVDFMap(&buf, root)
	return buf.Bytes(), appIDOf(root[0].Child[i]), true, nil
}

// AddShortcut adds a non-Steam game to the current account's library, or updates the entry that already is sc, and
// writes its art either way. Steam rewrites shortcuts.vdf on exit, so the caller makes sure Steam is closed. It
// reports whether the library changed (false when the same entry was already there).
func (s Steam) AddShortcut(sc Shortcut) (bool, error) {
	dir, err := s.userConfigDir()
	if err != nil {
		return false, err
	}
	path := filepath.Join(dir, "shortcuts.vdf")
	body, err := fsx.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, io.EOF) {
		return false, err
	}
	next, appID, changed, err := addShortcut(body, sc)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, writeGrid(dir, appID, sc.Cover)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return false, err
	}
	if err := backupBeforeEdit(path, next, 0o600); err != nil {
		return false, err
	}
	if err := datadir.WriteFile(path, next, 0o600); err != nil {
		return false, err
	}
	if err := writeGrid(dir, appID, sc.Cover); err != nil {
		return true, err
	}
	return true, nil
}

// removeShortcuts drops every entry of a shortcuts.vdf body that match selects and renumbers the rest, since Steam
// reads the list by consecutive index. It reports how many it dropped and returns body unchanged when none.
func removeShortcuts(body []byte, match func(exe, launchOptions string) bool) ([]byte, int, error) {
	root, err := readVDFMap(bufio.NewReader(bytes.NewReader(body)))
	if err != nil {
		return nil, 0, fmt.Errorf("read shortcuts.vdf: %w", err)
	}
	if len(root) != 1 || root[0].Kind != vdfMap || root[0].Key != "shortcuts" {
		return nil, 0, errors.New("shortcuts.vdf has no shortcuts list")
	}
	var kept []vdfNode
	for _, e := range root[0].Child {
		if match(field(e, "Exe"), field(e, "LaunchOptions")) {
			continue
		}
		e.Key = strconv.Itoa(len(kept))
		kept = append(kept, e)
	}
	removed := len(root[0].Child) - len(kept)
	if removed == 0 {
		return body, 0, nil
	}
	root[0].Child = kept
	var buf bytes.Buffer
	writeVDFMap(&buf, root)
	return buf.Bytes(), removed, nil
}

// RemoveShortcuts removes the shortcuts that run exe with launch options starting with optionsPrefix from every
// account's library, and reports how many it removed. Steam rewrites shortcuts.vdf on exit, so the caller makes sure
// Steam is closed.
func (s Steam) RemoveShortcuts(exe, optionsPrefix string) (int, error) {
	files, err := filepath.Glob(filepath.Join(s.Root, "userdata", "*", "config", "shortcuts.vdf"))
	if err != nil {
		return 0, err
	}
	want := quoted(exe)
	match := func(e, opts string) bool {
		return strings.EqualFold(e, want) && strings.HasPrefix(opts, optionsPrefix)
	}
	total := 0
	var errs []error
	for _, path := range files {
		body, err := fsx.ReadFile(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		next, n, err := removeShortcuts(body, match)
		if err == nil && n > 0 {
			if err = backupBeforeEdit(path, next, 0o600); err == nil {
				err = datadir.WriteFile(path, next, 0o600)
			}
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		total += n
	}
	return total, errors.Join(errs...)
}
