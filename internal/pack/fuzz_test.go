package pack

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func fuzzZip(files map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for n, c := range files {
		w, _ := zw.Create(n)
		_, _ = w.Write([]byte(c))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func FuzzParseR2Zip(f *testing.F) {
	f.Add(fuzzZip(map[string]string{exportFile: "profileName: p\ncommunity: c\nmods:\n- name: A-B-1.0.0\n  enabled: true\n", "config/a.cfg": "x"}))
	f.Add(fuzzZip(map[string]string{exportFile: ":\n- ["}))
	f.Add(fuzzZip(map[string]string{"../evil": "x"}))
	f.Add([]byte("PK"))
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := parseR2Zip(data)
		if err != nil {
			return
		}
		for _, fl := range append(append([]File{}, d.Configs...), d.Loose...) {
			if strings.HasPrefix(fl.Path, "/") || fl.Path == ".." || strings.HasPrefix(fl.Path, "../") {
				t.Fatalf("path escapes: %q", fl.Path)
			}
		}
	})
}

func FuzzModpackManifest(f *testing.F) {
	f.Add(fuzzZip(map[string]string{"manifest.json": `{"name":"n","version_number":"1.0.0","dependencies":["A-B-1.0.0"]}`}))
	f.Add(fuzzZip(map[string]string{"manifest.json": `{`}))
	f.Fuzz(func(t *testing.T, data []byte) {
		files, err := readZip(data)
		if err != nil {
			return
		}
		for name := range files {
			if strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") {
				t.Fatalf("path escapes: %q", name)
			}
		}
	})
}
