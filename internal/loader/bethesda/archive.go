package bethesda

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

func listArchive(path string) ([]string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".bsa":
		return listBSA(path)
	case ".ba2":
		return listBA2(path)
	default:
		return nil, fmt.Errorf("not a Bethesda archive")
	}
}

func listBSA(path string) ([]string, error) {
	f, err := safefs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	hdr := make([]byte, 36)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return nil, err
	}
	if string(hdr[:4]) != "BSA\x00" {
		return nil, fmt.Errorf("%s is not a BSA", path)
	}
	folderCount := binary.LittleEndian.Uint32(hdr[16:20])
	fileCount := binary.LittleEndian.Uint32(hdr[20:24])
	nameLen := binary.LittleEndian.Uint32(hdr[28:32])
	// Skip folder records (16 bytes each in v104; v105 adds 8).
	version := binary.LittleEndian.Uint32(hdr[4:8])
	rec := uint32(16)
	if version >= 105 {
		rec = 24
	}
	if _, err := f.Seek(int64(rec)*int64(folderCount), io.SeekCurrent); err != nil {
		return nil, err
	}
	// Folder names are length-prefixed; skip them and the per-folder file records
	// until the concatenated file-name blob.
	for range folderCount {
		var n [1]byte
		if _, err := io.ReadFull(f, n[:]); err != nil {
			return nil, err
		}
		if _, err := f.Seek(int64(n[0]), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
	if _, err := f.Seek(int64(fileCount)*16, io.SeekCurrent); err != nil {
		return nil, err
	}
	names := make([]byte, nameLen)
	if _, err := io.ReadFull(f, names); err != nil {
		return nil, err
	}
	var out []string
	for n := range strings.SplitSeq(strings.TrimRight(string(names), "\x00"), "\x00") {
		if n != "" {
			out = append(out, n)
		}
	}
	if len(out) != int(fileCount) && fileCount != 0 {
		return nil, fmt.Errorf("%s name count %d, header %d", path, len(out), fileCount)
	}
	return out, nil
}

func listBA2(path string) ([]string, error) {
	f, err := safefs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	hdr := make([]byte, 24)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return nil, err
	}
	if string(hdr[:4]) != "BTDX" {
		return nil, fmt.Errorf("%s is not a BA2", path)
	}
	fileCount := binary.LittleEndian.Uint32(hdr[12:16])
	nameOff := binary.LittleEndian.Uint64(hdr[16:24])
	if nameOff > math.MaxInt64 {
		return nil, fmt.Errorf("%s name table offset out of range", path)
	}
	if _, err := f.Seek(int64(nameOff), io.SeekStart); err != nil {
		return nil, err
	}
	out := make([]string, 0, fileCount)
	for range fileCount {
		var n uint16
		if err := binary.Read(f, binary.LittleEndian, &n); err != nil {
			return nil, err
		}
		b := make([]byte, n)
		if _, err := io.ReadFull(f, b); err != nil {
			return nil, err
		}
		out = append(out, string(b))
	}
	return out, nil
}
