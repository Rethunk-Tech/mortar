package backup

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/saves"
)

// Snap is one save folder inside a backup zip.
type Snap struct {
	Folder string `json:"folder"`
	Farm   string `json:"farm"`
}

// Backup is one zip in the backups folder.
type Backup struct {
	Name    string `json:"name"`
	At      int64  `json:"at"`
	Size    int64  `json:"size"`
	Profile string `json:"profile"`
	Kind    string `json:"kind"`
	Pinned  bool   `json:"pinned"`
	Saves   []Snap `json:"saves"`
}

// List returns backups newest first, each listing the saves it holds as l lays them out. A zip that cannot be opened
// still appears, with no save list.
func List(backupsDir string, l saves.Layout) ([]Backup, error) {
	names, err := list(backupsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Backup, 0, len(names))
	for _, name := range slices.Backward(names) {
		p := filepath.Join(backupsDir, name)
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		b := Backup{Name: name, Size: info.Size()}
		if t, err := time.Parse(stamp, strings.TrimSuffix(name, ".zip")); err == nil {
			b.At = t.UnixMilli()
		}
		c := readCause(p)
		b.Profile, b.Kind, b.Pinned = c.Profile, c.Kind, c.Pinned
		b.Saves, _ = snapsIn(p, l)
		out = append(out, b)
	}
	return out, nil
}

func snapsIn(zipPath string, l saves.Layout) ([]Snap, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	farms := map[string]string{}
	var order []string
	for _, f := range zr.File {
		folder, rest, ok := savePath(f.Name)
		if !ok {
			continue
		}
		if len(l.Files) > 0 {
			// A file save's name is its whole path under Saves/; its companions are not saves of their own.
			if folder, rest = path.Join(folder, rest), ""; f.FileInfo().IsDir() || !l.Matches(folder) {
				continue
			}
		}
		// A save that names no farm (Lethal Company's files) is left unnamed, for the reader to name by its folder.
		if _, seen := farms[folder]; !seen {
			farms[folder] = ""
			order = append(order, folder)
		}
		if rest == "SaveGameInfo" && !f.FileInfo().IsDir() {
			if name := farmName(f); name != "" {
				farms[folder] = name
			}
		}
	}
	slices.Sort(order)
	out := make([]Snap, 0, len(order))
	for _, folder := range order {
		out = append(out, Snap{Folder: folder, Farm: farms[folder]})
	}
	return out, nil
}

func savePath(name string) (folder, rest string, ok bool) {
	n := filepath.ToSlash(name)
	parts := strings.Split(n, "/")
	if len(parts) < 2 || parts[0] != "Saves" || parts[1] == "" || parts[1] == ".." {
		return "", "", false
	}
	return parts[1], strings.Join(parts[2:], "/"), true
}

type farmerInfo struct {
	FarmName string `xml:"farmName"`
}

func farmName(f *zip.File) string {
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	var info farmerInfo
	if xml.Unmarshal(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF}), &info) != nil {
		return ""
	}
	return strings.TrimSpace(info.FarmName)
}
