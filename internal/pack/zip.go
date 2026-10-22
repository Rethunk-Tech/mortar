package pack

import (
	"archive/zip"
	"bytes"
	"maps"
	"slices"
)

// Zip stores files in order, so the same files always make the same archive.
func Zip(files []File) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.Path)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SortedFiles is files by path.
func SortedFiles(files map[string][]byte) []File {
	out := make([]File, 0, len(files))
	for _, name := range slices.Sorted(maps.Keys(files)) {
		out = append(out, File{Path: name, Data: files[name]})
	}
	return out
}
