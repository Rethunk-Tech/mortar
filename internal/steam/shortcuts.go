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
	"strconv"

	"github.com/Rethunk-AI/mortar/internal/fsx"
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

// addShortcut appends sc to a shortcuts.vdf body (empty for a new file) unless an entry with the same command is
// already there, and reports whether it added one.
func addShortcut(body []byte, sc Shortcut) ([]byte, bool, error) {
	root := []vdfNode{{Kind: vdfMap, Key: "shortcuts"}}
	if len(body) > 0 {
		var err error
		if root, err = readVDFMap(bufio.NewReader(bytes.NewReader(body))); err != nil {
			return nil, false, fmt.Errorf("read shortcuts.vdf: %w", err)
		}
	}
	if len(root) != 1 || root[0].Kind != vdfMap || root[0].Key != "shortcuts" {
		return nil, false, errors.New("shortcuts.vdf has no shortcuts list")
	}
	for _, e := range root[0].Child {
		if field(e, "Exe") == quoted(sc.Exe) && field(e, "LaunchOptions") == sc.LaunchOptions {
			return body, false, nil
		}
	}
	root[0].Child = append(root[0].Child, sc.node(len(root[0].Child)))
	var buf bytes.Buffer
	writeVDFMap(&buf, root)
	return buf.Bytes(), true, nil
}

// AddShortcut adds a non-Steam game to the current account's library. Steam rewrites shortcuts.vdf on exit, so
// the caller makes sure Steam is closed. It reports whether an entry was added (false when it already exists).
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
	next, added, err := addShortcut(body, sc)
	if err != nil || !added {
		return false, err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return false, err
	}
	return true, fsx.WriteFile(path, next, 0o600)
}
